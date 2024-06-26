package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/syndicut/timonify/pkg/config"
)

const helpText = `Helmify parses kubernetes resources from std.in and converts it to a Helm module.

Example 1: 'kustomize build <kustomize_dir> | timonify mymodule' 
  - will create 'mymodule' directory with Helm module from kustomize output.

Example 2: 'cat my-app.yaml | timonify mymodule' 
  - will create 'mymodule' directory with Helm module from yaml file.

Example 3: 'timonify -f ./test_data/dir  mymodule' 
  - will scan directory ./test_data/dir for files with k8s manifests and create 'mymodule' directory with Helm module.

Example 4: 'timonify -f ./test_data/dir -r  mymodule' 
  - will scan directory ./test_data/dir recursively and  create 'mymodule' directory with Helm module.

Example 5: 'timonify -f ./test_data/dir -f ./test_data/sample-app.yaml -f ./test_data/dir/another_dir  mymodule' 
  - will scan provided multiple files and directories and  create 'mymodule' directory with Helm module.

Example 6: 'awk 'FNR==1 && NR!=1  {print "---"}{print}' /my_directory/*.yaml | timonify mymodule' 
  - will create 'mymodule' directory with Helm module from all yaml files in my_directory directory.

Usage:
  timonify [flags] CHART_NAME  -  CHART_NAME is optional. Default is 'module'. Can be a directory, e.g. 'deploy/modules/mymodule'.

Flags:
`

type arrayFlags []string

func (i *arrayFlags) String() string {
	if i == nil || len(*i) == 0 {
		return ""
	}
	return strings.Join(*i, ", ")
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// ReadFlags command-line flags into app config.
func ReadFlags() config.Config {
	files := arrayFlags{}
	result := config.Config{}
	var h, help, version, crd bool
	flag.BoolVar(&h, "h", false, "Print help. Example: timonify -h")
	flag.BoolVar(&help, "help", false, "Print help. Example: timonify -help")
	flag.BoolVar(&version, "version", false, "Print timonify version. Example: timonify -version")
	flag.BoolVar(&result.Verbose, "v", false, "Enable verbose output (print WARN & INFO). Example: timonify -v")
	flag.BoolVar(&result.VeryVerbose, "vv", false, "Enable very verbose output. Same as verbose but with DEBUG. Example: timonify -vv")
	flag.BoolVar(&crd, "crd-dir", false, "Enable crd install into 'crds' directory.\nWarning: CRDs placed in 'crds' directory will not be templated by Helm.\nSee https://helm.sh/docs/module_best_practices/custom_resource_definitions/#some-caveats-and-explanations\nExample: timonify -crd-dir")
	flag.BoolVar(&result.ImagePullSecrets, "image-pull-secrets", false, "Allows the user to use existing secrets as imagePullSecrets in values.yaml")
	flag.BoolVar(&result.GenerateDefaults, "generate-defaults", false, "Allows the user to add empty placeholders for typical customization options in values.yaml. Currently covers: topology constraints, node selectors, tolerances")
	flag.BoolVar(&result.CertManagerAsSubmodule, "cert-manager-as-submodule", false, "Allows the user to add cert-manager as a submodule")
	flag.StringVar(&result.CertManagerVersion, "cert-manager-version", "v1.12.2", "Allows the user to specify cert-manager submodule version. Only useful with cert-manager-as-submodule.")
	flag.BoolVar(&result.FilesRecursively, "r", false, "Scan dirs from -f option recursively")
	flag.BoolVar(&result.OriginalName, "original-name", false, "Use the object's original name instead of adding the module's release name as the common prefix.")
	flag.Var(&files, "f", "File or directory containing k8s manifests")

	flag.Parse()
	if h || help {
		fmt.Print(helpText)
		flag.PrintDefaults()
		os.Exit(0)
	}
	if version {
		printVersion()
		os.Exit(0)
	}
	name := flag.Arg(0)
	if name != "" {
		result.ModuleName = filepath.Base(name)
		result.ModuleDir = filepath.Dir(name)
	}
	if crd {
		result.Crd = crd
	}
	result.Files = files
	return result
}
