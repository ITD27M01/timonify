package timonify

import (
	"cuelang.org/go/cue/ast"
	"io"

	"github.com/syndicut/timonify/pkg/config"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Processor - converts k8s object to helm template.
// Implement this interface and register it to a context to support a new k8s resource conversion.
type Processor interface {
	// Process - converts k8s object to Helm template.
	// return false if not able to process given object type.
	Process(appMeta AppMetadata, unstructured *unstructured.Unstructured) (bool, Template, error)
}

// Template - represents Helm template in 'templates' directory.
type Template interface {
	// Filename - returns template filename
	Filename() string
	// Values - returns set of values used in template
	Values() *Values
	// Write - writes helm template into given writer
	Write(writer io.Writer) error
	// ObjectType - object type for config.cue file
	ObjectType() ast.Expr
	// ObjectLabel - object label for config.cue file
	ObjectLabel() ast.Label
}

// Output - converts Template into helm module on disk.
type Output interface {
	Create(moduleName, moduleDir string, Crd bool, templates []Template, filenames []string) error
}

// AppMetadata handle common information about K8s objects in the module.
type AppMetadata interface {
	// Namespace returns app namespace.
	Namespace() string
	// ModuleName returns module name
	ModuleName() string
	// TemplatedName converts object name to templated Helm name.
	// Example: 	"my-app-service1"	-> "{{ include "module.fullname" . }}-service1"
	//				"my-app-secret"		-> "{{ include "module.fullname" . }}-secret"
	//				etc...
	TemplatedName(objName string) string
	// TemplatedString converts a string to templated string with module name.
	TemplatedString(str string) string
	// TrimName trims common prefix from object name if exists.
	// We trim common prefix because helm already using release for this purpose.
	TrimName(objName string) string

	Config() config.Config
}
