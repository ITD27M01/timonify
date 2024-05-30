package timonify

import (
	"cuelang.org/go/cue/ast"
	"dario.cat/mergo"
	"fmt"
	"strings"

	"github.com/iancoleman/strcase"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Values - represents timoni values.
type Values struct {
	// Config represents values for config.cue file.
	Config *ast.StructLit
	// Values represents values for values.cue file.
	Values map[string]interface{}
}

func NewValues() *Values {
	return &Values{
		Config: ast.NewStruct(),
		Values: make(map[string]interface{}),
	}
}

// Merge given values with current instance.
func (v *Values) Merge(values *Values) error {
	if err := mergo.Merge(&v.Values, values.Values, mergo.WithAppendSlice); err != nil {
		return fmt.Errorf("%w: unable to merge timoni values", err)
	}
	mergeStructLits(v.Config, values.Config)

	return nil
}

func (v *Values) AddConfig(config ast.Expr, name ...string) error {
	name = toCamelCase(name)
	err := setNestedCueField(v.Config, config, name...)
	if err != nil {
		return fmt.Errorf("%w: unable to set nested cue field: %v", err, name)
	}
	return nil
}

// Add - adds given value to values and returns its timoni representation #config.<valueName>
func (v *Values) Add(config ast.Expr, value interface{}, name ...string) (string, error) {
	name = toCamelCase(name)
	switch val := value.(type) {
	case int:
		value = int64(val)
	case int8:
		value = int64(val)
	case int16:
		value = int64(val)
	case int32:
		value = int64(val)
	}

	err := v.AddConfig(config, name...)
	if err != nil {
		return "", fmt.Errorf("%w: unable to set config value: %v", err, name)
	}

	switch value := value.(type) {
	case []string:
		err = unstructured.SetNestedStringSlice(v.Values, value, name...)
	case map[string]string:
		err = unstructured.SetNestedStringMap(v.Values, value, name...)
	default:
		err = unstructured.SetNestedField(v.Values, value, name...)
	}
	if err != nil {
		return "", fmt.Errorf("%w: unable to set value: %v", err, name)
	}
	//_, isString := value.(string)
	//if isString {
	//	return "{{ .Values." + strings.Join(name, ".") + " | quote }}", nil
	//}
	//_, isSlice := value.([]interface{})
	//if isSlice {
	//	spaces := strconv.Itoa(len(name) * 2)
	//	return "{{ toYaml .Values." + strings.Join(name, ".") + " | nindent " + spaces + " }}", nil
	//}
	return "#config." + strings.Join(name, "."), nil
}

// setNestedCueField sets value inside ast.Node structure creating nested fields if needed from name
func setNestedCueField(config ast.Node, value ast.Expr, name ...string) error {
	// Start from the config node
	currentNode := config

	// Iterate over the name slice, but stop before the last element
	for _, n := range name[:len(name)-1] {
		// Try to find a field with the current name
		field := findField(currentNode, n)
		if field == nil {
			// If the field does not exist, create a new one
			field = &ast.Field{Label: ast.NewIdent(n), Value: &ast.StructLit{}}
			// Add the new field to the current node
			currentNode.(*ast.StructLit).Elts = append(currentNode.(*ast.StructLit).Elts, field)
		}
		// Move to the next node
		currentNode = field.Value
	}

	// Add the value to the last field
	lastField := findField(currentNode, name[len(name)-1])
	if lastField == nil {
		lastField = &ast.Field{Label: ast.NewIdent(name[len(name)-1]), Value: value}
		currentNode.(*ast.StructLit).Elts = append(currentNode.(*ast.StructLit).Elts, lastField)
	} else {
		lastField.Value = value
	}

	return nil
}

func findField(node ast.Node, name string) *ast.Field {
	if structLit, ok := node.(*ast.StructLit); ok {
		for _, elt := range structLit.Elts {
			if field, ok := elt.(*ast.Field); ok && field.Label.(*ast.Ident).Name == name {
				return field
			}
		}
	}
	return nil
}

// AddYaml - adds given value to values and returns its helm template representation as Yaml {{ .Values.<valueName> | toYaml | indent i }}
// indent  <= 0 will be omitted.
func (v *Values) AddYaml(value interface{}, indent int, newLine bool, name ...string) (string, error) {
	name = toCamelCase(name)
	err := unstructured.SetNestedField(v.Values, value, name...)
	if err != nil {
		return "", fmt.Errorf("%w: unable to set value: %v", err, name)
	}
	if indent > 0 {
		if newLine {
			return "{{ .Values." + strings.Join(name, ".") + fmt.Sprintf(" | toYaml | nindent %d }}", indent), nil
		}
		return "{{ .Values." + strings.Join(name, ".") + fmt.Sprintf(" | toYaml | indent %d }}", indent), nil
	}
	return "{{ .Values." + strings.Join(name, ".") + " | toYaml }}", nil
}

// AddSecret - adds empty value to values and returns its helm template representation {{ required "<valueName>" .Values.<valueName> }}.
// Set toBase64=true for Secret data to be base64 encoded and set false for Secret stringData.
func (v *Values) AddSecret(toBase64 bool, name ...string) (string, error) {
	name = toCamelCase(name)
	nameStr := strings.Join(name, ".")
	err := unstructured.SetNestedField(v.Values, "", name...)
	if err != nil {
		return "", fmt.Errorf("%w: unable to set value: %v", err, nameStr)
	}
	res := fmt.Sprintf(`{{ required "%[1]s is required" .Values.%[1]s`, nameStr)
	if toBase64 {
		res += " | b64enc"
	}
	return res + " | quote }}", err
}

func toCamelCase(name []string) []string {
	for i, n := range name {
		camelCase := strcase.ToLowerCamel(n)
		if n == strings.ToUpper(n) {
			camelCase = strcase.ToLowerCamel(strings.ToLower(n))
		}
		name[i] = camelCase
	}
	return name
}

func mergeStructLits(struct1, struct2 *ast.StructLit) *ast.StructLit {
	for _, elt2 := range struct2.Elts {
		field2, ok := elt2.(*ast.Field)
		if !ok {
			continue
		}
		found := false
		for _, elt1 := range struct1.Elts {
			field1, ok := elt1.(*ast.Field)
			if !ok {
				continue
			}
			if field1.Label.(*ast.Ident).Name == field2.Label.(*ast.Ident).Name {
				found = true
				if struct1, ok := field1.Value.(*ast.StructLit); ok {
					if struct2, ok := field2.Value.(*ast.StructLit); ok {
						field1.Value = mergeStructLits(struct1, struct2)
					}
				}
				break
			}
		}
		if !found {
			struct1.Elts = append(struct1.Elts, field2)
		}
	}
	return struct1
}
