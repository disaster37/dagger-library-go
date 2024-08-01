package main

import (
	"dagger/helm/internal/dagger"

	"github.com/creasty/defaults"
	"github.com/disaster37/dagger-library-go/lib/helper"
	"github.com/gookit/validate"
)

type GenerateSchemaOption struct {
	Source     *dagger.Directory `validate:"required"`
	FileName   string            `default:"values.schema.json"`
	ConfigFile string
}

// GenerateSchema permit to generate helm schema
// It will return the values.schema.json file
func (m *Helm) GenerateSchema(
	// the source directory
	source *dagger.Directory,

	// Config file for readme-generator
	// +optional
	configFile string,
) (schemaFile *dagger.File, err error) {

	option := &GenerateSchemaOption{
		Source:     source,
		ConfigFile: configFile,
	}

	if err = defaults.Set(option); err != nil {
		return nil, err
	}

	if err = validate.Struct(option).ValidateErr(); err != nil {
		return nil, err
	}

	container := m.BaseGeneratorContainer.
		WithDirectory("/project", source).
		WithWorkdir("/project")

	if option.ConfigFile == "" {
		container = container.
			WithExec(helper.ForgeCommandf("readme-generator -s %s --values values.yaml", option.FileName))
	} else {
		container = container.
			WithExec(helper.ForgeCommandf("readme-generator -c %s -s %s --values values.yaml", option.ConfigFile, option.FileName))
	}

	schemaFile = container.File(option.FileName)

	return schemaFile, nil
}
