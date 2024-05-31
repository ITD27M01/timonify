package processor

import (
	"cuelang.org/go/cue/ast"
	"io"
	"strings"

	"github.com/sirupsen/logrus"
	cueformat "github.com/syndicut/timonify/pkg/cue"
	"github.com/syndicut/timonify/pkg/timonify"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var nsGVK = schema.GroupVersionKind{
	Group:   "",
	Version: "v1",
	Kind:    "Namespace",
}

// Default default processor for unknown resources.
func Default() timonify.Processor {
	return &dft{}
}

type dft struct{}

// Process unknown resource to a helm template. Default processor just templates obj name and adds helm annotations.
func (d dft) Process(appMeta timonify.AppMetadata, obj *unstructured.Unstructured) (bool, timonify.Template, error) {
	if obj.GroupVersionKind() == nsGVK {
		// Skip namespaces from processing because namespace will be handled by Helm.
		return true, nil, nil
	}
	logrus.WithFields(logrus.Fields{
		"ApiVersion": obj.GetAPIVersion(),
		"Kind":       obj.GetKind(),
		"Name":       obj.GetName(),
	}).Warn("Unsupported resource: using default processor.")
	name := appMeta.TrimName(obj.GetName())
	label := strings.ToLower(obj.GetKind())

	meta, err := ProcessObjMeta(appMeta, obj)
	if err != nil {
		return true, nil, err
	}
	delete(obj.Object, "apiVersion")
	delete(obj.Object, "kind")
	delete(obj.Object, "metadata")

	body, err := cueformat.Marshal(obj.Object, 0)
	if err != nil {
		return true, nil, err
	}
	return true, &defaultResult{
		data:  []byte(meta + "\n" + body),
		name:  name,
		label: label,
	}, nil
}

type defaultResult struct {
	data  []byte
	name  string
	label string
}

func (r *defaultResult) Filename() string {
	return r.name + ".cue"
}

func (r *defaultResult) Values() *timonify.Values {
	return timonify.NewValues()
}

func (r *defaultResult) Write(writer io.Writer) error {
	_, err := writer.Write(r.data)
	return err
}

func (r *defaultResult) ObjectType() ast.Expr {
	return ast.NewIdent("_")
}

func (r *defaultResult) ObjectLabel() ast.Label {
	return ast.NewIdent(r.label)
}
