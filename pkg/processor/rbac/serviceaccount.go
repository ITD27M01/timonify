package rbac

import (
	"github.com/syndicut/timonify/pkg/processor"
	"github.com/syndicut/timonify/pkg/timonify"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var serviceAccountGVC = schema.GroupVersionKind{
	Group:   "",
	Version: "v1",
	Kind:    "ServiceAccount",
}

// ServiceAccount creates processor for k8s ServiceAccount resource.
func ServiceAccount() timonify.Processor {
	return &serviceAccount{}
}

type serviceAccount struct{}

// Process k8s ServiceAccount object into helm template. Returns false if not capable of processing given resource type.
func (sa serviceAccount) Process(appMeta timonify.AppMetadata, obj *unstructured.Unstructured) (bool, timonify.Template, error) {
	if obj.GroupVersionKind() != serviceAccountGVC {
		return false, nil, nil
	}
	values := timonify.Values{}
	meta, err := processor.ProcessObjMeta(appMeta, obj, processor.WithAnnotations(values))
	if err != nil {
		return true, nil, err
	}

	return true, &saResult{
		data:   []byte(meta),
		values: values,
	}, nil
}

type saResult struct {
	data   []byte
	values timonify.Values
}

func (r *saResult) Filename() string {
	return "serviceaccount.yaml"
}

func (r *saResult) Values() timonify.Values {
	return r.values
}

func (r *saResult) Write(writer io.Writer) error {
	_, err := writer.Write(r.data)
	return err
}
