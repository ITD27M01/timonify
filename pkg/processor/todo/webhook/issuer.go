package webhook

import (
	"bytes"
	"fmt"
	cueformat "github.com/syndicut/timonify/pkg/cue"
	"io"

	"github.com/syndicut/timonify/pkg/timonify"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

const (
	issuerTempl = `apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: {{ include "%[1]s.fullname" . }}-%[2]s
  labels:
  {{- include "%[1]s.labels" . | nindent 4 }}
spec:
%[3]s`
	issuerTemplWithAnno = `apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: {{ include "%[1]s.fullname" . }}-%[2]s
  annotations:
    "helm.sh/hook": post-install,post-upgrade
    "helm.sh/hook-weight": "1"
  labels:
  {{- include "%[1]s.labels" . | nindent 4 }}
spec:
%[3]s`
)

var issuerGVC = schema.GroupVersionKind{
	Group:   "cert-manager.io",
	Version: "v1",
	Kind:    "Issuer",
}

// Issuer creates processor for k8s Issuer resource.
func Issuer() timonify.Processor {
	return &issuer{}
}

type issuer struct{}

// Process k8s Issuer object into template. Returns false if not capable of processing given resource type.
func (i issuer) Process(appMeta timonify.AppMetadata, obj *unstructured.Unstructured) (bool, timonify.Template, error) {
	if obj.GroupVersionKind() != issuerGVC {
		return false, nil, nil
	}
	name := appMeta.TrimName(obj.GetName())
	spec, _ := yaml.Marshal(obj.Object["spec"])
	spec = cueformat.Indent(spec, 2)
	spec = bytes.TrimRight(spec, "\n ")
	tmpl := ""
	if appMeta.Config().CertManagerAsSubchart {
		tmpl = issuerTemplWithAnno
	} else {
		tmpl = issuerTempl
	}
	res := fmt.Sprintf(tmpl, appMeta.ChartName(), name, string(spec))
	return true, &issResult{
		name: name,
		data: []byte(res),
	}, nil
}

type issResult struct {
	name string
	data []byte
}

func (r *issResult) Filename() string {
	return r.name + ".yaml"
}

func (r *issResult) Values() *timonify.Values {
	return timonify.NewValues()
}

func (r *issResult) Write(writer io.Writer) error {
	_, err := writer.Write(r.data)
	return err
}
