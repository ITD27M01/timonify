package processor

import (
	"cuelang.org/go/cue/ast"
	cueformat "cuelang.org/go/cue/format"
	"fmt"
	"github.com/iancoleman/strcase"
	"io"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/syndicut/timonify/pkg/cue"
	"github.com/syndicut/timonify/pkg/timonify"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var nsGVK = schema.GroupVersionKind{
	Group:   "",
	Version: "v1",
	Kind:    "Namespace",
}

var defaultTmpl = `package templates

%s: {
	#config: #Config
	%s
}
`

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

	meta, err := ProcessObjMeta(appMeta, obj)
	if err != nil {
		return true, nil, err
	}
	delete(obj.Object, "apiVersion")
	delete(obj.Object, "kind")
	delete(obj.Object, "metadata")

	body, err := cue.Marshal(obj.Object, 0, false)
	if err != nil {
		return true, nil, err
	}
	body = strings.Trim(body, "{}")
	return true, &defaultResult{
		data: []byte(meta + "\n" + body),
		name: name,
	}, nil
}

type defaultResult struct {
	data []byte
	name string
}

func (r *defaultResult) Filename() string {
	return r.name + ".cue"
}

func (r *defaultResult) Values() *timonify.Values {
	return timonify.NewValues()
}

func (r *defaultResult) Write(writer io.Writer) error {
	formatted, err := cueformat.Source([]byte(fmt.Sprintf(defaultTmpl, r.ObjectType(), r.data)))
	if err != nil {
		return fmt.Errorf("failed to format cue: %w", err)
	}
	_, err = writer.Write(formatted)
	return err
}

func (r *defaultResult) ObjectType() ast.Expr {
	return ast.NewIdent("#" + strcase.ToCamel(r.name))
}

func (r *defaultResult) ObjectLabel() ast.Label {
	return ast.NewIdent(strings.ReplaceAll(r.name, "-", ""))
}
