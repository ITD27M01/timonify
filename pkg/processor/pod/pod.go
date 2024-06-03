package pod

import (
	"fmt"
	"strconv"
	"strings"

	"cuelang.org/go/cue/ast"
	"github.com/iancoleman/strcase"
	"github.com/syndicut/timonify/pkg/cluster"
	cueformat "github.com/syndicut/timonify/pkg/cue"
	securityContext "github.com/syndicut/timonify/pkg/processor/security-context"
	"github.com/syndicut/timonify/pkg/timonify"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

const imagePullPolicyTemplate = "#config.%[1]s.%[2]s.imagePullPolicy"
const envValue = "#config.%[1]s.%[2]s.%[3]s.%[4]s"

func ProcessSpec(objName string, appMeta timonify.AppMetadata, spec corev1.PodSpec) (map[string]interface{}, *timonify.Values, error) {
	values, err := processPodSpec(objName, appMeta, &spec)
	if err != nil {
		return nil, nil, err
	}

	// replace PVC to templated name
	for i := 0; i < len(spec.Volumes); i++ {
		vol := spec.Volumes[i]
		if vol.PersistentVolumeClaim == nil {
			continue
		}
		tempPVCName := appMeta.TemplatedName(vol.PersistentVolumeClaim.ClaimName)

		spec.Volumes[i].PersistentVolumeClaim.ClaimName = tempPVCName
	}

	// replace container resources with template to values.
	specMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&spec)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: unable to convert podSpec to map", err)
	}

	specMap, values, err = processNestedContainers(specMap, objName, values, "containers")
	if err != nil {
		return nil, nil, err
	}

	specMap, values, err = processNestedContainers(specMap, objName, values, "initContainers")
	if err != nil {
		return nil, nil, err
	}

	if appMeta.Config().ImagePullSecrets {
		if _, defined := specMap["imagePullSecrets"]; !defined {
			specMap["imagePullSecrets"] = "{{ .Values.imagePullSecrets | default list | toJson }}"
			if _, err := values.Add(ast.NewSel(ast.NewIdent("corev1"), "#LocalObjectReference"), []string{}, "imagePullSecrets"); err != nil {
				return nil, nil, err
			}
		}
	}

	err = securityContext.ProcessContainerSecurityContext(objName, specMap, values)
	if err != nil {
		return nil, nil, err
	}

	// process nodeSelector if presented:
	if spec.NodeSelector != nil {
		err = unstructured.SetNestedField(specMap, fmt.Sprintf(`#config.%s.nodeSelector`, objName), "nodeSelector")
		if err != nil {
			return nil, nil, err
		}
		_, err = values.Add(ast.NewStruct(&ast.Field{
			Label: ast.NewList(ast.NewIdent("string")),
			Value: ast.NewIdent("string"),
		}), spec.NodeSelector, objName, "nodeSelector")
		if err != nil {
			return nil, nil, err
		}
	}

	return specMap, values, nil
}

func processNestedContainers(specMap map[string]interface{}, objName string, values *timonify.Values, containerKey string) (map[string]interface{}, *timonify.Values, error) {
	containers, _, err := unstructured.NestedSlice(specMap, containerKey)
	if err != nil {
		return nil, nil, err
	}

	if len(containers) > 0 {
		containers, values, err = processContainers(objName, *values, containerKey, containers)
		if err != nil {
			return nil, nil, err
		}

		err = unstructured.SetNestedSlice(specMap, containers, containerKey)
		if err != nil {
			return nil, nil, err
		}
	}

	return specMap, values, nil
}

func processContainers(objName string, values timonify.Values, containerType string, containers []interface{}) ([]interface{}, *timonify.Values, error) {
	for i := range containers {
		containerName := strcase.ToLowerCamel((containers[i].(map[string]interface{})["name"]).(string))
		res, exists, err := unstructured.NestedMap(values.Values, objName, containerName, "resources")
		if err != nil {
			return nil, nil, err
		}
		if exists && len(res) > 0 {
			err = unstructured.SetNestedField(containers[i].(map[string]interface{}), fmt.Sprintf(`#config.%s.%s.resources`, objName, containerName), "resources")
			if err != nil {
				return nil, nil, err
			}
		}

		args, exists, err := unstructured.NestedStringSlice(containers[i].(map[string]interface{}), "args")
		if err != nil {
			return nil, nil, err
		}
		if exists && len(args) > 0 {
			err = unstructured.SetNestedField(containers[i].(map[string]interface{}), fmt.Sprintf(`#config.%[1]s.%[2]s.args`, objName, containerName), "args")
			if err != nil {
				return nil, nil, err
			}

			_, err = values.Add(ast.NewList(&ast.Ellipsis{Type: ast.NewIdent("string")}), args, objName, containerName, "args")
			if err != nil {
				return nil, nil, fmt.Errorf("%w: unable to set deployment value field", err)
			}
		}
	}
	return containers, &values, nil
}

func processPodSpec(name string, appMeta timonify.AppMetadata, pod *corev1.PodSpec) (*timonify.Values, error) {
	values := timonify.NewValues()
	for i, c := range pod.Containers {
		processed, err := processPodContainer(name, appMeta, c, values)
		if err != nil {
			return nil, err
		}
		pod.Containers[i] = processed
	}

	for i, c := range pod.InitContainers {
		processed, err := processPodContainer(name, appMeta, c, values)
		if err != nil {
			return nil, err
		}
		pod.InitContainers[i] = processed
	}

	for _, v := range pod.Volumes {
		if v.ConfigMap != nil {
			v.ConfigMap.Name = appMeta.TemplatedName(v.ConfigMap.Name)
		}
		if v.Secret != nil {
			v.Secret.SecretName = appMeta.TemplatedName(v.Secret.SecretName)
		}
	}
	pod.ServiceAccountName = appMeta.TemplatedName(pod.ServiceAccountName)

	for i, s := range pod.ImagePullSecrets {
		pod.ImagePullSecrets[i].Name = appMeta.TemplatedName(s.Name)
	}

	return values, nil
}

func processPodContainer(name string, appMeta timonify.AppMetadata, c corev1.Container, values *timonify.Values) (corev1.Container, error) {
	image, err := strconv.Unquote(c.Image)
	if err != nil {
		return c, fmt.Errorf("%w: unable to unquote image", err)
	}
	index := strings.LastIndex(image, ":")
	var isDigest bool
	if strings.Contains(image, "@") && strings.Count(image, ":") >= 2 {
		last := strings.LastIndex(c.Image, ":")
		index = strings.LastIndex(c.Image[:last], ":")
		isDigest = true
	}
	if index < 0 {
		return c, fmt.Errorf("wrong image format: %q", image)
	}
	repo := image[:index]
	var tag, digest string
	if isDigest {
		digest = image[index+1:]
	} else {
		tag = image[index+1:]
	}
	containerName := strcase.ToLowerCamel(c.Name)
	c.Image = fmt.Sprintf("#config.%[1]s.%[2]s.image.reference", name, containerName)

	if _, err := values.Add(nil, strconv.Quote(repo), name, containerName, "image", "repository"); err != nil {
		return c, fmt.Errorf("%w: unable to set deployment value field", err)
	}
	if _, err := values.Add(nil, strconv.Quote(tag), name, containerName, "image", "tag"); err != nil {
		return c, fmt.Errorf("%w: unable to set deployment value field", err)
	}
	if _, err := values.Add(nil, strconv.Quote(digest), name, containerName, "image", "digest"); err != nil {
		return c, fmt.Errorf("%w: unable to set deployment value field", err)
	}

	// Image has a common timoni schema, so we override it here
	err = values.AddConfig(ast.NewSel(ast.NewIdent("timoniv1"), "#Image"), true, name, containerName, "image")
	if err != nil {
		return c, fmt.Errorf("%w: unable to set image value field", err)
	}

	c, err = processEnv(name, appMeta, c, values)
	if err != nil {
		return c, err
	}

	for _, e := range c.EnvFrom {
		if e.SecretRef != nil {
			e.SecretRef.Name = appMeta.TemplatedName(e.SecretRef.Name)
		}
		if e.ConfigMapRef != nil {
			e.ConfigMapRef.Name = appMeta.TemplatedName(e.ConfigMapRef.Name)
		}
	}
	c.Env = append(c.Env, corev1.EnvVar{
		Name:  strconv.Quote(cluster.DomainEnv),
		Value: fmt.Sprintf("#config.%s", cluster.DomainKey),
	})
	for k, v := range c.Resources.Requests {
		_, err = values.Add(ast.NewSel(ast.NewIdent("timoniv1"), "#ResourceList"), strconv.Quote(v.String()), name, containerName, "resources", "requests", k.String())
		if err != nil {
			return c, fmt.Errorf("%w: unable to set container resources value", err)
		}
	}
	for k, v := range c.Resources.Limits {
		_, err = values.Add(ast.NewSel(ast.NewIdent("timoniv1"), "#ResourceList"), strconv.Quote(v.String()), name, containerName, "resources", "limits", k.String())
		if err != nil {
			return c, fmt.Errorf("%w: unable to set container resources value", err)
		}
	}
	resourcesSchema := cueformat.MustParse(`timoniv1.#ResourceRequirements & {
		requests: {
			cpu:    *"10m" | timoniv1.#CPUQuantity
			memory: *"32Mi" | timoniv1.#MemoryQuantity
		}
	}`)
	if err := values.AddConfig(resourcesSchema, false, name, containerName, "resources"); err != nil {
		return c, fmt.Errorf("%w: unable to add resources config value", err)
	}

	if c.ImagePullPolicy != "" {
		_, err = values.Add(ast.NewSel(ast.NewIdent("corev1"), "#PullPolicy"), string(c.ImagePullPolicy), name, containerName, "imagePullPolicy")
		if err != nil {
			return c, fmt.Errorf("%w: unable to set container imagePullPolicy", err)
		}
		c.ImagePullPolicy = corev1.PullPolicy(fmt.Sprintf(imagePullPolicyTemplate, name, containerName))
	}
	return c, nil
}

func processEnv(name string, appMeta timonify.AppMetadata, c corev1.Container, values *timonify.Values) (corev1.Container, error) {
	containerName := strcase.ToLowerCamel(c.Name)
	for i := 0; i < len(c.Env); i++ {
		if c.Env[i].ValueFrom != nil {
			switch {
			case c.Env[i].ValueFrom.SecretKeyRef != nil:
				c.Env[i].ValueFrom.SecretKeyRef.Name = appMeta.TemplatedName(c.Env[i].ValueFrom.SecretKeyRef.Name)
			case c.Env[i].ValueFrom.ConfigMapKeyRef != nil:
				c.Env[i].ValueFrom.ConfigMapKeyRef.Name = appMeta.TemplatedName(c.Env[i].ValueFrom.ConfigMapKeyRef.Name)
			case c.Env[i].ValueFrom.FieldRef != nil, c.Env[i].ValueFrom.ResourceFieldRef != nil:
				// nothing to change here, keep the original value
			}
			continue
		}

		_, err := values.Add(ast.NewIdent("string"), c.Env[i].Value, name, containerName, "env", strcase.ToLowerCamel(strings.ToLower(c.Env[i].Name)))
		if err != nil {
			return c, fmt.Errorf("%w: unable to set deployment value field", err)
		}
		c.Env[i].Value = fmt.Sprintf(envValue, name, containerName, "env", strcase.ToLowerCamel(strings.ToLower(c.Env[i].Name)))
	}
	return c, nil
}
