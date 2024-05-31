package deployment

import (
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/token"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/syndicut/timonify/pkg/processor/pod"

	"github.com/iancoleman/strcase"
	cueformat "github.com/syndicut/timonify/pkg/cue"
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
	timoniv1 "timoni.sh/core/v1alpha1"
)

#Deployment: appsv1.#Deployment & {
	#config:    #Config
{{- .Meta }}
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
			spec: corev1.#PodSpec & {
{{ .Spec }}
			}
		}
	}
}`)

const selectorTempl = `selector: matchLabels: #config.selector.labels & %[1]s
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

	matchLabels, err := cueformat.Marshal(depl.Spec.Selector.MatchLabels, 0)
	if err != nil {
		return true, nil, err
	}
	matchExpr := ""
	if depl.Spec.Selector.MatchExpressions != nil {
		matchExpr, err = cueformat.Marshal(map[string]interface{}{
			"selector": map[string]interface{}{
				"matchExpressions": depl.Spec.Selector.MatchExpressions,
			},
		}, 4)
		if err != nil {
			return true, nil, err
		}
	}
	selector := fmt.Sprintf(selectorTempl, matchLabels, matchExpr)
	selector = strings.Trim(selector, " \n")
	selector = string(cueformat.Indent([]byte(selector), 4))

	podLabels, err := cueformat.Marshal(depl.Spec.Template.ObjectMeta.Labels, 0)
	if err != nil {
		return true, nil, err
	}
	podLabels = fmt.Sprintf("#config.selector.labels & %s", podLabels)

	podAnnotations := ""
	if len(depl.Spec.Template.ObjectMeta.Annotations) != 0 {
		podAnnotations, err = cueformat.Marshal(map[string]interface{}{"annotations": depl.Spec.Template.ObjectMeta.Annotations}, 6)
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

	spec, err := cueformat.Marshal(specMap, 6)
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

var replicasSchema = &ast.BinaryExpr{
	Op: token.OR,
	X: &ast.UnaryExpr{
		Op: token.MUL,
		X:  ast.NewLit(token.INT, "1"),
	},
	Y: &ast.BinaryExpr{
		Op: token.AND,
		X:  ast.NewIdent("int"),
		Y: &ast.BinaryExpr{
			Op: token.GTR,
			X:  ast.NewIdent("int"),
			Y:  ast.NewLit(token.INT, "0"),
		},
	},
}

func processReplicas(name string, deployment *appsv1.Deployment, values *timonify.Values) (string, error) {
	if deployment.Spec.Replicas == nil {
		return "", nil
	}
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
	revisionHistoryLimit, err := cueformat.Marshal(map[string]interface{}{"revisionHistoryLimit": revisionHistoryLimitTpl}, 2)
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
	//var buf bytes.Buffer
	//if err := deploymentTempl.Execute(&buf, r.data); err != nil {
	//	return fmt.Errorf("failed to execute template: %w", err)
	//}
	//formatted, err := format.Source(buf.Bytes())
	//if err != nil {
	//	return fmt.Errorf("failed to format cue: %w", err)
	//}
	//_, err = writer.Write(formatted)
	//return err
	return deploymentTempl.Execute(writer, r.data)
}

func (r *result) ObjectType() ast.Expr {
	return ast.NewIdent("#Deployment")
}

func (r *result) ObjectLabel() ast.Label {
	return ast.NewIdent("deploy")
}
