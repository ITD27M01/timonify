package timoni

import (
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/token"
	"fmt"
	"github.com/syndicut/timonify/pkg/cluster"
	cueformat "github.com/syndicut/timonify/pkg/cue"
	"github.com/syndicut/timonify/pkg/timonify"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

const defaultValues = `// Code generated by timoni.
// Note that this file must have no imports and all values must be concrete.

@if(!debug)

package main

// Defaults
values: %s
`

// NewOutput creates interface to dump processed input to filesystem in timoni module format.
func NewOutput() timonify.Output {
	return &output{}
}

type output struct{}

// Create a timoni module in the current directory:
// moduleName/
//
//	├── cue.mod
//	│   ├── gen # Kubernetes APIs and CRDs schemas
//	│   ├── pkg # Timoni APIs schemas
//	│   └── module.cue # Module metadata
//	├── templates
//	│   ├── config.cue # Config schema and default values
//	├── timoni.cue # Timoni entry point
//	├── timoni.ignore # Timoni ignore rules
//	├── values.cue # Timoni values placeholder
//	└── README.md # Module documentation
//
// Overwrites existing values.cue and templates in templates dir on every run.
func (o output) Create(moduleDir, chartName string, crd bool, templates []timonify.Template, filenames []string) error {
	err := initModuleDir(moduleDir, chartName, crd)
	if err != nil {
		return err
	}
	// group templates into files
	files := map[string][]timonify.Template{}
	values := timonify.NewValues()
	if _, err := values.Add(ast.NewIdent("string"), cluster.DefaultDomain, cluster.DomainKey); err != nil {
		return fmt.Errorf("%w: unable to set domain value", err)
	}
	for i, template := range templates {
		file := files[filenames[i]]
		file = append(file, template)
		files[filenames[i]] = file
		err = values.Merge(template.Values())
		if err != nil {
			return err
		}
	}
	cDir := filepath.Join(moduleDir, chartName)
	for filename, tpls := range files {
		err = overwriteTemplateFile(filename, cDir, tpls)
		if err != nil {
			return err
		}
	}
	err = overwriteValuesFile(cDir, values)
	if err != nil {
		return err
	}
	err = overwriteConfigFile(cDir, values, files)
	if err != nil {
		return err
	}
	return nil
}

func overwriteTemplateFile(filename, chartDir string, templates []timonify.Template) error {
	subdir := "templates"
	file := filepath.Join(chartDir, subdir, filename)
	f, err := os.OpenFile(file, os.O_APPEND|os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("%w: unable to open %s", err, file)
	}
	defer f.Close()
	for i, t := range templates {
		logrus.WithField("file", file).Debug("writing a template into")
		err = t.Write(f)
		if err != nil {
			return fmt.Errorf("%w: unable to write into %s", err, file)
		}
		if i != len(templates)-1 {
			_, err = f.Write([]byte("\n---\n"))
			if err != nil {
				return fmt.Errorf("%w: unable to write into %s", err, file)
			}
		}
	}
	logrus.WithField("file", file).Info("overwritten")
	return nil
}

func overwriteValuesFile(chartDir string, values *timonify.Values) error {
	res, err := cueformat.Marshal(values.Values, 0)
	if err != nil {
		return fmt.Errorf("%w: unable to write marshal values.cue", err)
	}

	file := filepath.Join(chartDir, "values.cue")
	err = os.WriteFile(file, []byte(fmt.Sprintf(defaultValues, res)), 0600)
	if err != nil {
		return fmt.Errorf("%w: unable to write values.cue", err)
	}
	logrus.WithField("file", file).Info("overwritten")
	return nil
}

func overwriteConfigFile(chartDir string, values *timonify.Values, files map[string][]timonify.Template) error {
	objectsNode := ast.NewStruct()
	for _, templates := range files {
		for _, t := range templates {
			objectsNode.Elts = append(objectsNode.Elts,
				&ast.Field{
					Label: t.ObjectLabel(),
					Value: &ast.BinaryExpr{
						Op: token.AND, // Represents the '&' operator
						X:  t.ObjectType(),
						Y: &ast.StructLit{
							Elts: []ast.Decl{
								&ast.Field{
									Label: ast.NewIdent("#config"),
									Value: ast.NewIdent("config"),
								},
							},
						},
					},
				},
			)
		}
	}

	file := filepath.Join(chartDir, "templates", "config.cue")
	b, err := format.Node(defaultConfig(objectsNode, values.Config.Elts...))
	if err != nil {
		return fmt.Errorf("%w: unable to format config.cue", err)
	}
	err = os.WriteFile(file, b, 0600)
	if err != nil {
		return fmt.Errorf("%w: unable to write config.cue", err)
	}
	logrus.WithField("file", file).Info("overwritten")
	return nil
}
