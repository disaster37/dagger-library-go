package main

import (
	"dagger/helm/internal/dagger"

	"github.com/disaster37/dagger-library-go/lib/v2/helper"
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

	if err = validatePathArg("targetFile", targetFile); err != nil {
		return nil, err
	}
	if err = validatePathArg("configFile", configFile); err != nil {
		return nil, err
	}

	container = container.
		WithExec(helper.ForgeCommandf("readme-generator schema%s -v values.yaml -s %s", configArg(configFile), targetFile))
	schemaFile = container.File(targetFile)

	return schemaFile, nil
}
