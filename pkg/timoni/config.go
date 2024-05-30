package timoni

import (
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/token"
)

func defaultConfig(objects *ast.StructLit, schema ...ast.Decl) *ast.File {
	// Create a new file
	file := &ast.File{}

	// Add package name
	file.Decls = append(file.Decls, &ast.Package{Name: ast.NewIdent("templates")})

	// Add imports
	file.Decls = append(file.Decls, &ast.ImportDecl{
		Specs: []*ast.ImportSpec{
			{Name: ast.NewIdent("corev1"), Path: &ast.BasicLit{Kind: token.STRING, Value: "\"k8s.io/api/core/v1\""}},
			{Name: ast.NewIdent("timoniv1"), Path: &ast.BasicLit{Kind: token.STRING, Value: "\"timoni.sh/core/v1alpha1\""}},
		},
	})

	// Add Config field
	configField := &ast.Field{
		Label: ast.NewIdent("#Config"),
		Value: ast.NewStruct(
			// kubeVersion field
			&ast.Field{
				Label: ast.NewIdent("kubeVersion"),
				Value: ast.NewIdent("string"),
			},
			// clusterVersion field
			&ast.Field{
				Label: ast.NewIdent("clusterVersion"),
				Value: &ast.BinaryExpr{
					Op: token.AND,
					X:  ast.NewSel(ast.NewIdent("timoniv1"), "#SemVer"),
					Y: &ast.StructLit{
						Elts: []ast.Decl{
							&ast.Field{
								Label: ast.NewIdent("#Version"),
								Value: ast.NewIdent("kubeVersion"),
							},
							&ast.Field{
								Label: ast.NewIdent("#Minimum"),
								Value: &ast.BasicLit{Kind: token.STRING, Value: "\"1.20.0\""},
							},
						},
					},
				},
			},
			// moduleVersion field
			&ast.Field{
				Label: ast.NewIdent("moduleVersion"),
				Value: ast.NewIdent("string"),
			},
			// metadata field
			&ast.Field{
				Label: ast.NewIdent("metadata"),
				Value: &ast.BinaryExpr{
					Op: token.AND,
					X:  ast.NewSel(ast.NewIdent("timoniv1"), "#Metadata"),
					Y: &ast.StructLit{
						Elts: []ast.Decl{
							&ast.Field{
								Label: ast.NewIdent("#Version"),
								Value: ast.NewIdent("moduleVersion"),
							},
						},
					},
				},
			},
			// metadata: labels field
			&ast.Field{
				Label: ast.NewIdent("metadata"),
				Value: &ast.StructLit{
					Elts: []ast.Decl{
						&ast.Field{
							Label: ast.NewIdent("labels"),
							Value: &ast.BinaryExpr{
								Op: token.AND,
								X:  ast.NewSel(ast.NewIdent("timoniv1"), "#Labels"),
								Y: &ast.StructLit{
									Elts: []ast.Decl{
										&ast.Field{
											Label: &ast.BasicLit{Kind: token.STRING, Value: "\"app.kubernetes.io/created-by\""},
											Value: ast.NewSel(ast.NewIdent("metadata"), "name"),
										},
										&ast.Field{
											Label: &ast.BasicLit{Kind: token.STRING, Value: "\"\\(timoniv1.#StdLabelPartOf)\""},
											Value: ast.NewSel(ast.NewIdent("metadata"), "name"),
										},
									},
								},
							},
						},
					},
				},
			},
			// metadata: annotations field
			&ast.Field{
				Label: ast.NewIdent("metadata"),
				Value: &ast.StructLit{
					Elts: []ast.Decl{
						&ast.Field{
							Label: ast.NewIdent("annotations"),
							Value: ast.NewSel(ast.NewIdent("timoniv1"), "#Annotations"),
						},
					},
				},
			},
		),
	}
	configField.Value.(*ast.StructLit).Elts = append(configField.Value.(*ast.StructLit).Elts, schema...)
	file.Decls = append(file.Decls, configField)

	// Add Instance field
	instanceField := &ast.Field{
		Label: ast.NewIdent("#Instance"),
		Value: &ast.StructLit{
			Elts: []ast.Decl{
				// config field
				&ast.Field{
					Label: ast.NewIdent("config"),
					Value: ast.NewIdent("#Config"),
				},
				// objects field
				&ast.Field{
					Label: ast.NewIdent("objects"),
					Value: objects,
				},
			},
		},
	}
	file.Decls = append(file.Decls, instanceField)

	return file
}
