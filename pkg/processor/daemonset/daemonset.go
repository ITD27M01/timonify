package daemonset

import (
	"fmt"
	cueformat "github.com/syndicut/timonify/pkg/cue"
	"github.com/syndicut/timonify/pkg/processor/pod"
	"io"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
	"github.com/syndicut/timonify/pkg/processor"
	"github.com/syndicut/timonify/pkg/timonify"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var daemonsetGVC = schema.GroupVersionKind{
	Group:   "apps",
	Version: "v1",
	Kind:    "DaemonSet",
}

var daemonsetTempl, _ = template.New("daemonset").Parse(
	`{{- .Meta }}
spec:
  selector:
{{ .Selector }}
  template:
    metadata:
      labels:
{{ .PodLabels }}
{{- .PodAnnotations }}
    spec:
{{ .Spec }}`)

const selectorTempl = `%[1]s
{{- include "%[2]s.selectorLabels" . | nindent 6 }}
%[3]s`

// New creates processor for k8s Daemonset resource.
func New() timonify.Processor {
	return &daemonset{}
}

type daemonset struct{}

// Process k8s Daemonset object into template. Returns false if not capable of processing given resource type.
func (d daemonset) Process(appMeta timonify.AppMetadata, obj *unstructured.Unstructured) (bool, timonify.Template, error) {
	if obj.GroupVersionKind() != daemonsetGVC {
		return false, nil, nil
	}
	dae := appsv1.DaemonSet{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &dae)
	if err != nil {
		return true, nil, fmt.Errorf("%w: unable to cast to daemonset", err)
	}
	meta, err := processor.ProcessObjMeta(appMeta, obj)
	if err != nil {
		return true, nil, err
	}

	values := timonify.Values{}

	name := appMeta.TrimName(obj.GetName())

	matchLabels, err := cueformat.Marshal(map[string]interface{}{"matchLabels": dae.Spec.Selector.MatchLabels}, 0)
	if err != nil {
		return true, nil, err
	}
	matchExpr := ""
	if dae.Spec.Selector.MatchExpressions != nil {
		matchExpr, err = cueformat.Marshal(map[string]interface{}{"matchExpressions": dae.Spec.Selector.MatchExpressions}, 0)
		if err != nil {
			return true, nil, err
		}
	}
	selector := fmt.Sprintf(selectorTempl, matchLabels, appMeta.ChartName(), matchExpr)
	selector = strings.Trim(selector, " \n")
	selector = string(cueformat.Indent([]byte(selector), 4))

	podLabels, err := cueformat.Marshal(dae.Spec.Template.ObjectMeta.Labels, 8)
	if err != nil {
		return true, nil, err
	}
	podLabels += fmt.Sprintf("\n      {{- include \"%s.selectorLabels\" . | nindent 8 }}", appMeta.ChartName())

	podAnnotations := ""
	if len(dae.Spec.Template.ObjectMeta.Annotations) != 0 {
		podAnnotations, err = cueformat.Marshal(map[string]interface{}{"annotations": dae.Spec.Template.ObjectMeta.Annotations}, 6)
		if err != nil {
			return true, nil, err
		}

		podAnnotations = "\n" + podAnnotations
	}

	nameCamel := strcase.ToLowerCamel(name)
	specMap, podValues, err := pod.ProcessSpec(nameCamel, appMeta, dae.Spec.Template.Spec)
	if err != nil {
		return true, nil, err
	}
	err = values.Merge(podValues)
	if err != nil {
		return true, nil, err
	}

	spec, err := cueformat.Marshal(specMap, 6)
	if err != nil {
		return true, nil, err
	}
	spec = strings.ReplaceAll(spec, "'", "")

	return true, &result{
		values: values,
		data: struct {
			Meta           string
			Selector       string
			PodLabels      string
			PodAnnotations string
			Spec           string
		}{
			Meta:           meta,
			Selector:       selector,
			PodLabels:      podLabels,
			PodAnnotations: podAnnotations,
			Spec:           spec,
		},
	}, nil
}

type result struct {
	data struct {
		Meta           string
		Selector       string
		PodLabels      string
		PodAnnotations string
		Spec           string
	}
	values timonify.Values
}

func (r *result) Filename() string {
	return "daemonset.yaml"
}

func (r *result) Values() timonify.Values {
	return r.values
}

func (r *result) Write(writer io.Writer) error {
	return daemonsetTempl.Execute(writer, r.data)
}
