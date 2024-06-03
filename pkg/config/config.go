package config

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/validation"
)

// defaultModuleName - default name for a helm module directory.
const defaultModuleName = "timoni"

// Config for Helmify application.
type Config struct {
	// ModuleName name of the Timoni module and its base directory where timoni.cue is located.
	ModuleName string
	// ModuleDir - optional path to module dir. Full module path will be: ModuleDir/ModuleName/timoni.cue.
	ModuleDir string
	// Verbose set true to see WARN and INFO logs.
	Verbose bool
	// VeryVerbose set true to see WARN, INFO, and DEBUG logs.
	VeryVerbose bool
	// crd-dir set true to enable crd folder.
	Crd bool
	// ImagePullSecrets flag
	ImagePullSecrets bool
	// GenerateDefaults enables the generation of empty values placeholders for common customization options of helm module
	// current generated values: tolerances, node selectors, topology constraints
	GenerateDefaults bool
	// CertManagerAsSubmodule enables the generation of a submodule for cert-manager
	CertManagerAsSubmodule bool
	// CertManagerVersion sets cert-manager version in dependency
	CertManagerVersion string
	// Files - directories or files with k8s manifests
	Files []string
	// FilesRecursively read Files recursively
	FilesRecursively bool
	// OriginalName retains Kubernetes resource's original name
	OriginalName bool
}

func (c *Config) Validate() error {
	if c.ModuleName == "" {
		logrus.Infof("Module name is not set. Using default name '%s", defaultModuleName)
		c.ModuleName = defaultModuleName
	}
	err := validation.IsDNS1123Subdomain(c.ModuleName)
	if err != nil {
		for _, e := range err {
			logrus.Errorf("Invalid module name %s", e)
		}
		return fmt.Errorf("invalid module name %s", c.ModuleName)
	}
	return nil
}
