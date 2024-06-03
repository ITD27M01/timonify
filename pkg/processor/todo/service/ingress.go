package service

import (
	"fmt"
	cueformat "github.com/syndicut/timonify/pkg/cue"
	"github.com/syndicut/timonify/pkg/processor"
	"github.com/syndicut/timonify/pkg/timonify"
	"io"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"text/template"
)

var ingressTempl, _ = template.New("ingress").Parse(
	`{{ .Meta }}
{{ .Spec }}`)

var ingressGVC = schema.GroupVersionKind{
	Group:   "networking.k8s.io",
	Version: "v1",
	Kind:    "Ingress",
}

// NewIngress creates processor for k8s Ingress resource.
func NewIngress() timonify.Processor {
	return &ingress{}
}

type ingress struct{}

// Process k8s Service object into template. Returns false if not capable of processing given resource type.
func (r ingress) Process(appMeta timonify.AppMetadata, obj *unstructured.Unstructured) (bool, timonify.Template, error) {
	if obj.GroupVersionKind() != ingressGVC {
		return false, nil, nil
	}
	ing := networkingv1.Ingress{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &ing)
	if err != nil {
		return true, nil, fmt.Errorf("%w: unable to cast to ingress", err)
	}
	meta, err := processor.ProcessObjMeta(appMeta, obj)
	if err != nil {
		return true, nil, err
	}
	name := appMeta.TrimName(obj.GetName())
	processIngressSpec(appMeta, &ing.Spec)
	spec, err := cueformat.Marshal(map[string]interface{}{"spec": &ing.Spec}, 0, true)
	if err != nil {
		return true, nil, err
	}

	return true, &ingressResult{
		name: name + ".yaml",
		data: struct {
			Meta string
			Spec string
		}{Meta: meta, Spec: spec},
	}, nil
}

func processIngressSpec(appMeta timonify.AppMetadata, ing *networkingv1.IngressSpec) {
	if ing.DefaultBackend != nil && ing.DefaultBackend.Service != nil {
		ing.DefaultBackend.Service.Name = appMeta.TemplatedName(ing.DefaultBackend.Service.Name)
	}
	for i := range ing.Rules {
		if ing.Rules[i].IngressRuleValue.HTTP != nil {
			for j := range ing.Rules[i].IngressRuleValue.HTTP.Paths {
				if ing.Rules[i].IngressRuleValue.HTTP.Paths[j].Backend.Service != nil {
					ing.Rules[i].IngressRuleValue.HTTP.Paths[j].Backend.Service.Name = appMeta.TemplatedName(ing.Rules[i].IngressRuleValue.HTTP.Paths[j].Backend.Service.Name)
				}
			}
		}
	}
}

type ingressResult struct {
	name string
	data struct {
		Meta string
		Spec string
	}
}

func (r *ingressResult) Filename() string {
	return r.name
}

func (r *ingressResult) Values() *timonify.Values {
	return timonify.NewValues()
}

func (r *ingressResult) Write(writer io.Writer) error {
	return ingressTempl.Execute(writer, r.data)
}
