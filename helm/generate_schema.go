package main

import (
	"dagger/helm/internal/dagger"

	"github.com/disaster37/dagger-library-go/lib/helper"
)

// GenerateSchema permit to generate helm schema
// It will return the values.schema.json file
func (m *Helm) GenerateSchema(

	// The target file
	// +optional
	// +default="values.schema.json"
	targetFile string,

	// Config file for readme-generator
	// +optional
	configFile string,
) (schemaFile *dagger.File, err error) {

	container := m.GeneratorContainer
	if targetFile == "" {
		targetFile = "values.schema.json"
	}

	if configFile == "" {
		container = container.
			WithExec(helper.ForgeCommandf("readme-generator -s %s --values values.yaml", targetFile))
	} else {
		container = container.
			WithExec(helper.ForgeCommandf("readme-generator -c %s -s %s --values values.yaml", configFile, targetFile))
	}
	schemaFile = container.File(targetFile)
	m = m.WithSource(m.Src.WithFile(targetFile, schemaFile))

	return schemaFile, nil
}
