package rbac

import (
	"fmt"
	cueformat "github.com/syndicut/timonify/pkg/cue"
	"io"
	"strings"
	"text/template"

	"github.com/syndicut/timonify/pkg/processor"

	"github.com/syndicut/timonify/pkg/timonify"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var roleBindingTempl, _ = template.New("roleBinding").Parse(
	`{{- .Meta }}
{{ .RoleRef }}
{{ .Subjects }}`)

var roleBindingGVC = schema.GroupVersionKind{
	Group:   "rbac.authorization.k8s.io",
	Version: "v1",
	Kind:    "RoleBinding",
}

// RoleBinding creates processor for k8s RoleBinding resource.
func RoleBinding() timonify.Processor {
	return &roleBinding{}
}

type roleBinding struct{}

// Process k8s RoleBinding object into helm template. Returns false if not capable of processing given resource type.
func (r roleBinding) Process(appMeta timonify.AppMetadata, obj *unstructured.Unstructured) (bool, timonify.Template, error) {
	if obj.GroupVersionKind() != roleBindingGVC {
		return false, nil, nil
	}
	rb := rbacv1.RoleBinding{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &rb)
	if err != nil {
		return true, nil, fmt.Errorf("%w: unable to cast to RoleBinding", err)
	}
	meta, err := processor.ProcessObjMeta(appMeta, obj)
	if err != nil {
		return true, nil, err
	}

	rb.RoleRef.Name = appMeta.TemplatedName(rb.RoleRef.Name)

	roleRef, err := cueformat.Marshal(map[string]interface{}{"roleRef": &rb.RoleRef}, 0)
	if err != nil {
		return true, nil, err
	}

	for i, s := range rb.Subjects {
		s.Namespace = "{{ .Release.Namespace }}"
		s.Name = appMeta.TemplatedName(s.Name)
		rb.Subjects[i] = s
	}
	subjects, err := cueformat.Marshal(map[string]interface{}{"subjects": &rb.Subjects}, 0)
	if err != nil {
		return true, nil, err
	}

	return true, &rbResult{
		name: appMeta.TrimName(obj.GetName()),
		data: struct {
			Meta     string
			RoleRef  string
			Subjects string
		}{
			Meta:     meta,
			RoleRef:  roleRef,
			Subjects: subjects,
		},
	}, nil
}

type rbResult struct {
	name string
	data struct {
		Meta     string
		RoleRef  string
		Subjects string
	}
}

func (r *rbResult) Filename() string {
	return strings.TrimSuffix(r.name, "-rolebinding") + "-rbac.yaml"
}

func (r *rbResult) Values() timonify.Values {
	return timonify.Values{}
}

func (r *rbResult) Write(writer io.Writer) error {
	return roleBindingTempl.Execute(writer, r.data)
}
