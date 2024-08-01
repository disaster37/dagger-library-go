package main

import (
	"github.com/creasty/defaults"
	"dagger/helm/internal/dagger"
	"github.com/disaster37/dagger-library-go/lib/helper"
	"github.com/gookit/validate"
)

type GenerateDocumentationOption struct {
	Source     *dagger.Directory `validate:"required"`
	FileName   string            `default:"README.md"`
	ConfigFile string
}

// GenerateDocumentation permit to generate helm documentation
// It will return the readme file
func (m *Helm) GenerateDocumentation(

	// the source directory
	source *dagger.Directory,

	// Config file for readme-generator
	// +optional
	configFile string,
) (readmeFile *dagger.File, err error) {
	option := &GenerateDocumentationOption{
		Source:     source,
		ConfigFile: configFile,
	}

	if err = defaults.Set(option); err != nil {
		return nil, err
	}

	if err = validate.Struct(option).ValidateErr(); err != nil {
		return nil, err
	}

	container := m.baseGeneratorContainer.
		WithDirectory("/project", source).
		WithWorkdir("/project")

	if option.ConfigFile == "" {
		container = container.
			WithExec(helper.ForgeCommandf("readme-generator -r %s --values values.yaml", option.FileName))
	} else {
		container = container.
			WithExec(helper.ForgeCommandf("readme-generator -c %s -r %s --values values.yaml", option.ConfigFile, option.FileName))
	}

	readmeFile = container.File(option.FileName)

	return readmeFile, nil

}
