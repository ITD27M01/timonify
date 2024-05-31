package security_context

import (
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/token"
	"fmt"

	"github.com/iancoleman/strcase"
	"github.com/syndicut/timonify/pkg/timonify"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	sc             = "securityContext"
	cscValueName   = "containerSecurityContext"
	timoniTemplate = "#config.%[1]s.%[2]s.containerSecurityContext"
)

// ProcessContainerSecurityContext adds 'securityContext' to the podSpec in specMap, if it doesn't have one already defined.
func ProcessContainerSecurityContext(nameCamel string, specMap map[string]interface{}, values *timonify.Values) error {
	err := processSecurityContext(nameCamel, "containers", specMap, values)
	if err != nil {
		return err
	}

	err = processSecurityContext(nameCamel, "initContainers", specMap, values)
	if err != nil {
		return err
	}

	return nil
}

func processSecurityContext(nameCamel string, containerType string, specMap map[string]interface{}, values *timonify.Values) error {
	if containers, defined := specMap[containerType]; defined {
		for _, container := range containers.([]interface{}) {
			castedContainer := container.(map[string]interface{})
			containerName := strcase.ToLowerCamel(castedContainer["name"].(string))
			if _, defined2 := castedContainer["securityContext"]; defined2 {
				err := setSecContextValue(nameCamel, containerName, castedContainer, values)
				if err != nil {
					return err
				}
			}
		}
		err := unstructured.SetNestedField(specMap, containers, containerType)
		if err != nil {
			return err
		}
	}
	return nil
}

func setSecContextValue(resourceName string, containerName string, castedContainer map[string]interface{}, values *timonify.Values) error {
	if castedContainer["securityContext"] != nil {
		securityContextSchema := &ast.BinaryExpr{
			Op: token.AND, // Represents the '&' operator
			X:  ast.NewSel(ast.NewIdent("corev1"), "#SecurityContext"),
			Y: &ast.StructLit{
				Elts: []ast.Decl{
					&ast.Field{
						Label: ast.NewIdent("allowPrivilegeEscalation"),
						Value: ast.NewBool(false),
					},
					&ast.Field{
						Label: ast.NewIdent("capabilities"),
						Value: &ast.StructLit{
							Elts: []ast.Decl{
								&ast.Field{
									Label: ast.NewIdent("drop"),
									Value: &ast.BinaryExpr{
										Op: token.OR,
										X: &ast.ListLit{
											Elts: []ast.Expr{
												&ast.BasicLit{Kind: token.STRING, Value: "\"ALL\""},
											},
										},
										Y: ast.NewList(
											&ast.Ellipsis{
												Type: ast.NewIdent("string"),
											},
										),
									},
								},
							},
						},
					},
				},
			},
		}
		_, err := values.Add(securityContextSchema, castedContainer["securityContext"], resourceName, containerName, cscValueName)
		if err != nil {
			return err
		}

		valueString := fmt.Sprintf(timoniTemplate, resourceName, containerName)

		err = unstructured.SetNestedField(castedContainer, valueString, sc)
		if err != nil {
			return err
		}
	}
	return nil
}
