package processor

import (
	"fmt"
	"strings"

	"github.com/iancoleman/strcase"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	cueformat "github.com/syndicut/timonify/pkg/cue"
	"github.com/syndicut/timonify/pkg/timonify"
)

const metaTemplate = `apiVersion: %[1]s
kind:       %[2]s
metadata: {
  name: %[3]s
  labels: #config.%[4]s.metadata.labels & {
%[5]s
%[6]s`

const annotationsTemplate = `  if #config.%[1]s.%[2]s.podAnnotations != _|_ {
		annotations: #config.%[1]s.%[2]s.podAnnotations
	}`

type MetaOpt interface {
	apply(*options)
}

type options struct {
	values      timonify.Values
	annotations bool
}

type annotationsOption struct {
	values timonify.Values
}

func (a annotationsOption) apply(opts *options) {
	opts.annotations = true
	opts.values = a.values
}

func WithAnnotations(values timonify.Values) MetaOpt {
	return annotationsOption{
		values: values,
	}
}

// ProcessObjMeta - returns object apiVersion, kind and metadata as timoni template.
func ProcessObjMeta(appMeta timonify.AppMetadata, obj *unstructured.Unstructured, opts ...MetaOpt) (string, error) {
	options := &options{}
	for _, opt := range opts {
		opt.apply(options)
	}

	var err error
	var labels, annotations string
	if len(obj.GetLabels()) != 0 {
		l := obj.GetLabels()
		// provided by Timoni
		delete(l, "app.kubernetes.io/name")
		delete(l, "app.kubernetes.io/version")
		delete(l, "app.kubernetes.io/managed-by")

		// Since we delete labels above, it is possible that at this point there are no more labels.
		if len(l) > 0 {
			labels, err = cueformat.Marshal(l, 4)
			if err != nil {
				return "", err
			}
		}
	}
	if len(obj.GetAnnotations()) != 0 {
		annotations, err = cueformat.Marshal(map[string]interface{}{"annotations": obj.GetAnnotations()}, 2)
		if err != nil {
			return "", err
		}
	}

	templatedName := appMeta.TemplatedName(obj.GetName())
	apiVersion, kind := obj.GetObjectKind().GroupVersionKind().ToAPIVersionAndKind()

	var metaStr string
	if options.values != nil && options.annotations {
		name := strcase.ToLowerCamel(appMeta.TrimName(obj.GetName()))
		kind := strcase.ToLowerCamel(kind)
		valuesAnnotations := make(map[string]interface{})
		for k, v := range obj.GetAnnotations() {
			valuesAnnotations[k] = v
		}
		err = unstructured.SetNestedField(options.values, valuesAnnotations, name, kind, "annotations")
		if err != nil {
			return "", err
		}

		annotations = fmt.Sprintf(annotationsTemplate, name, kind)
	}

	metaStr = fmt.Sprintf(metaTemplate, apiVersion, kind, templatedName, appMeta.ChartName(), labels, annotations)
	metaStr = strings.Trim(metaStr, " \n")
	metaStr = strings.ReplaceAll(metaStr, "\n\n", "\n")
	return metaStr, nil
}
