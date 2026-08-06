package main

import (
	"dagger/helm/internal/dagger"

	"github.com/disaster37/dagger-library-go/lib/helper"
)

// GenerateDocumentation permit to generate helm documentation
// It will return the readme file
func (m *Helm) GenerateDocumentation(

	// The target file
	// +optional
	// +default="README.md"
	targetFile string,

	// Config file for readme-generator
	// +optional
	configFile string,
) (readmeFile *dagger.File, err error) {

	container := m.GeneratorContainer
	if targetFile == "" {
		targetFile = "README.md"
	}

	if err = validatePathArg("targetFile", targetFile); err != nil {
		return nil, err
	}
	if err = validatePathArg("configFile", configFile); err != nil {
		return nil, err
	}

	container = container.
		WithExec(helper.ForgeCommandf("readme-generator readme%s -v values.yaml -r %s", configArg(configFile), targetFile))
	readmeFile = container.File(targetFile)

	return readmeFile, nil

}
