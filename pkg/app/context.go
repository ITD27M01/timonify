package app

import (
	"github.com/sirupsen/logrus"
	"github.com/syndicut/timonify/pkg/config"
	"github.com/syndicut/timonify/pkg/metadata"
	"github.com/syndicut/timonify/pkg/timonify"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// appContext helm processing context. Stores processed objects.
type appContext struct {
	processors       []timonify.Processor
	defaultProcessor timonify.Processor
	output           timonify.Output
	config           config.Config
	appMeta          *metadata.Service
	objects          []*unstructured.Unstructured
	fileNames        []string
}

// New returns context with config set.
func New(config config.Config, output timonify.Output) *appContext {
	return &appContext{
		config:  config,
		appMeta: metadata.New(config),
		output:  output,
	}
}

// WithProcessors  add processors to the context and returns it.
func (c *appContext) WithProcessors(processors ...timonify.Processor) *appContext {
	c.processors = append(c.processors, processors...)
	return c
}

// WithDefaultProcessor  add defaultProcessor for unknown resources to the context and returns it.
func (c *appContext) WithDefaultProcessor(processor timonify.Processor) *appContext {
	c.defaultProcessor = processor
	return c
}

// Add k8s object to app context.
func (c *appContext) Add(obj *unstructured.Unstructured, filename string) {
	// we need to add all objects before start processing only to define app metadata.
	c.appMeta.Load(obj)
	c.objects = append(c.objects, obj)
	c.fileNames = append(c.fileNames, filename)
}

// CreateHelm creates helm module from context k8s objects.
func (c *appContext) CreateHelm(stop <-chan struct{}) error {
	logrus.WithFields(logrus.Fields{
		"ModuleName": c.appMeta.ModuleName(),
		"Namespace":  c.appMeta.Namespace(),
	}).Info("creating a module")
	var templates []timonify.Template
	var filenames []string
	for i, obj := range c.objects {
		template, err := c.process(obj)
		if err != nil {
			return err
		}
		if template != nil {
			templates = append(templates, template)
			filename := template.Filename()
			if c.fileNames[i] != "" {
				filename = c.fileNames[i]
			}
			filenames = append(filenames, filename)
		}
		select {
		case <-stop:
			return nil
		default:
		}
	}
	return c.output.Create(c.config.ModuleDir, c.config.ModuleName, c.config.Crd, templates, filenames)
}

func (c *appContext) process(obj *unstructured.Unstructured) (timonify.Template, error) {
	for _, p := range c.processors {
		if processed, result, err := p.Process(c.appMeta, obj); processed {
			if err != nil {
				return nil, err
			}
			logrus.WithFields(logrus.Fields{
				"ApiVersion": obj.GetAPIVersion(),
				"Kind":       obj.GetKind(),
				"Name":       obj.GetName(),
			}).Debug("processed")
			return result, nil
		}
	}
	if c.defaultProcessor == nil {
		logrus.WithFields(logrus.Fields{
			"ApiVersion": obj.GetAPIVersion(),
			"Kind":       obj.GetKind(),
			"Name":       obj.GetName(),
		}).Warn("Skipping: no suitable processor for resource.")
		return nil, nil
	}
	_, t, err := c.defaultProcessor.Process(c.appMeta, obj)
	return t, err
}
