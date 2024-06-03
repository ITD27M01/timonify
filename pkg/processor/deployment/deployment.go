package deployment

import (
	"bytes"
	"cuelang.org/go/cue/ast"
	"fmt"
	"github.com/syndicut/timonify/pkg/format"
	"io"
	"strings"
	"text/template"

	"github.com/syndicut/timonify/pkg/processor/pod"

	cueformat "cuelang.org/go/cue/format"
	"github.com/iancoleman/strcase"
	"github.com/syndicut/timonify/pkg/cue"
	"github.com/syndicut/timonify/pkg/processor"
	"github.com/syndicut/timonify/pkg/timonify"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var deploymentGVC = schema.GroupVersionKind{
	Group:   "apps",
	Version: "v1",
	Kind:    "Deployment",
}

// var deploymentTempl, _ = template.New("deployment").Parse(
//
//	`{{- .Meta }}
//
// spec:
// {{- if .Replicas }}
// {{ .Replicas }}
// {{- end }}
// {{- if .RevisionHistoryLimit }}
// {{ .RevisionHistoryLimit }}
// {{- end }}
//
//	selector:
//
// {{ .Selector }}
//
//	template:
//	  metadata:
//	    labels:
//
// {{ .PodLabels }}
// {{- .PodAnnotations }}
//
//	spec:
//
// {{ .Spec }}`)
var deploymentTempl, _ = template.New("deployment").Parse(
	`package templates

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

#Deployment: appsv1.#Deployment & {
	#config:    #Config
{{ .Meta }}
	spec: appsv1.#DeploymentSpec & {
{{- if .Replicas }}
{{ .Replicas }}
{{- end }}
{{- if .RevisionHistoryLimit }}
{{ .RevisionHistoryLimit }}
{{- end }}
{{ .Selector }}
		template: {
			metadata: {
				labels: {{ .PodLabels }}
{{- .PodAnnotations }}
			}
			spec: corev1.#PodSpec & {{ .Spec }}
		}
	}
}`)

const selectorTempl = `selector: matchLabels: %[1]s
%[2]s`

// New creates processor for k8s Deployment resource.
func New() timonify.Processor {
	return &deployment{}
}

type deployment struct{}

// Process k8s Deployment object into template. Returns false if not capable of processing given resource type.
func (d deployment) Process(appMeta timonify.AppMetadata, obj *unstructured.Unstructured) (bool, timonify.Template, error) {
	if obj.GroupVersionKind() != deploymentGVC {
		return false, nil, nil
	}
	depl := appsv1.Deployment{}

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &depl)
	if err != nil {
		return true, nil, fmt.Errorf("%w: unable to cast to deployment", err)
	}
	format.QuoteStringsInStruct(&depl)
	meta, err := processor.ProcessObjMeta(appMeta, obj)
	if err != nil {
		return true, nil, err
	}

	values := timonify.NewValues()

	name := appMeta.TrimName(obj.GetName())
	replicas, err := processReplicas(name, &depl, values)
	if err != nil {
		return true, nil, err
	}

	revisionHistoryLimit, err := processRevisionHistoryLimit(name, &depl, values)
	if err != nil {
		return true, nil, err
	}

	matchLabels, err := cue.Marshal(depl.Spec.Selector.MatchLabels, 0, true)
	if err != nil {
		return true, nil, err
	}
	matchExpr := ""
	if depl.Spec.Selector.MatchExpressions != nil {
		matchExpr, err = cue.Marshal(map[string]interface{}{
			"selector": map[string]interface{}{
				"matchExpressions": depl.Spec.Selector.MatchExpressions,
			},
		}, 4, true)
		if err != nil {
			return true, nil, err
		}
	}
	selector := fmt.Sprintf(selectorTempl, matchLabels, matchExpr)
	selector = strings.Trim(selector, " \n")
	selector = string(cue.Indent([]byte(selector), 4))

	podLabels, err := cue.Marshal(depl.Spec.Template.ObjectMeta.Labels, 0, true)
	if err != nil {
		return true, nil, err
	}

	podAnnotations := ""
	if len(depl.Spec.Template.ObjectMeta.Annotations) != 0 {
		podAnnotations, err = cue.Marshal(map[string]interface{}{"annotations": depl.Spec.Template.ObjectMeta.Annotations}, 6, true)
		if err != nil {
			return true, nil, err
		}

		podAnnotations = "\n" + podAnnotations
	}

	nameCamel := strcase.ToLowerCamel(name)
	specMap, podValues, err := pod.ProcessSpec(nameCamel, appMeta, depl.Spec.Template.Spec)
	if err != nil {
		return true, nil, err
	}
	err = values.Merge(podValues)
	if err != nil {
		return true, nil, err
	}

	spec, err := cue.Marshal(specMap, 6, true)
	if err != nil {
		return true, nil, err
	}

	spec = strings.ReplaceAll(spec, "'", "")

	return true, &result{
		values: values,
		data: struct {
			Meta                 string
			Replicas             string
			RevisionHistoryLimit string
			Selector             string
			PodLabels            string
			PodAnnotations       string
			Spec                 string
		}{
			Meta:                 meta,
			Replicas:             replicas,
			RevisionHistoryLimit: revisionHistoryLimit,
			Selector:             selector,
			PodLabels:            podLabels,
			PodAnnotations:       podAnnotations,
			Spec:                 spec,
		},
	}, nil
}

func processReplicas(name string, deployment *appsv1.Deployment, values *timonify.Values) (string, error) {
	if deployment.Spec.Replicas == nil {
		return "", nil
	}
	replicasSchema := cue.MustParse("*1 | int & >0")
	replicasTpl, err := values.Add(replicasSchema, int64(*deployment.Spec.Replicas), name, "replicas")
	if err != nil {
		return "", err
	}
	replicas := fmt.Sprintf("replicas: %s", replicasTpl)
	return replicas, nil
}

func processRevisionHistoryLimit(name string, deployment *appsv1.Deployment, values *timonify.Values) (string, error) {
	if deployment.Spec.RevisionHistoryLimit == nil {
		return "", nil
	}
	revisionHistoryLimitTpl, err := values.Add(ast.NewIdent("int64"), int64(*deployment.Spec.RevisionHistoryLimit), name, "revisionHistoryLimit")
	if err != nil {
		return "", err
	}
	revisionHistoryLimit, err := cue.Marshal(map[string]interface{}{"revisionHistoryLimit": revisionHistoryLimitTpl}, 2, true)
	if err != nil {
		return "", err
	}
	revisionHistoryLimit = strings.ReplaceAll(revisionHistoryLimit, "'", "")
	return revisionHistoryLimit, nil
}

type result struct {
	data struct {
		Meta                 string
		Replicas             string
		RevisionHistoryLimit string
		Selector             string
		PodLabels            string
		PodAnnotations       string
		Spec                 string
	}
	values *timonify.Values
}

func (r *result) Filename() string {
	return "deployment.cue"
}

func (r *result) Values() *timonify.Values {
	return r.values
}

func (r *result) Write(writer io.Writer) error {
	var buf bytes.Buffer
	if err := deploymentTempl.Execute(&buf, r.data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	formatted, err := cueformat.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to format cue: %w", err)
	}
	_, err = writer.Write(formatted)
	return err
}

func (r *result) ObjectType() ast.Expr {
	return ast.NewIdent("#Deployment")
}

func (r *result) ObjectLabel() ast.Label {
	return ast.NewIdent("deploy")
}
