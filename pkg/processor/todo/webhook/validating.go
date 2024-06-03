package webhook

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/syndicut/timonify/pkg/timonify"
	v1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

const (
	vwhTempl = `apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: {{ include "%[1]s.fullname" . }}-%[2]s
  annotations:
    cert-manager.io/inject-ca-from: {{ .Release.Namespace }}/{{ include "%[1]s.fullname" . }}-%[3]s
  labels:
  {{- include "%[1]s.labels" . | nindent 4 }}
webhooks:
%[4]s`
)

var vwhGVK = schema.GroupVersionKind{
	Group:   "admissionregistration.k8s.io",
	Version: "v1",
	Kind:    "ValidatingWebhookConfiguration",
}

// ValidatingWebhook creates processor for k8s ValidatingWebhookConfiguration resource.
func ValidatingWebhook() timonify.Processor {
	return &vwh{}
}

type vwh struct{}

// Process k8s ValidatingWebhookConfiguration object into template. Returns false if not capable of processing given resource type.
func (w vwh) Process(appMeta timonify.AppMetadata, obj *unstructured.Unstructured) (bool, timonify.Template, error) {
	if obj.GroupVersionKind() != vwhGVK {
		return false, nil, nil
	}
	name := appMeta.TrimName(obj.GetName())

	whConf := v1.ValidatingWebhookConfiguration{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &whConf)
	if err != nil {
		return true, nil, fmt.Errorf("%w: unable to cast to ValidatingWebhookConfiguration", err)
	}
	for i, whc := range whConf.Webhooks {
		whc.ClientConfig.Service.Name = appMeta.TemplatedName(whc.ClientConfig.Service.Name)
		whc.ClientConfig.Service.Namespace = strings.ReplaceAll(whc.ClientConfig.Service.Namespace, appMeta.Namespace(), `{{ .Release.Namespace }}`)
		whConf.Webhooks[i] = whc
	}
	webhooks, _ := yaml.Marshal(whConf.Webhooks)
	webhooks = bytes.TrimRight(webhooks, "\n ")
	certName, _, err := unstructured.NestedString(obj.Object, "metadata", "annotations", "cert-manager.io/inject-ca-from")
	if err != nil {
		return true, nil, fmt.Errorf("%w: unable get webhook certName", err)
	}
	certName = strings.TrimPrefix(certName, appMeta.Namespace()+"/")
	certName = appMeta.TrimName(certName)
	res := fmt.Sprintf(vwhTempl, appMeta.ModuleName(), name, certName, string(webhooks))
	return true, &vwhResult{
		name: name,
		data: []byte(res),
	}, nil
}

type vwhResult struct {
	name string
	data []byte
}

func (r *vwhResult) Filename() string {
	return r.name + ".yaml"
}

func (r *vwhResult) Values() *timonify.Values {
	return timonify.NewValues()
}

func (r *vwhResult) Write(writer io.Writer) error {
	_, err := writer.Write(r.data)
	return err
}
