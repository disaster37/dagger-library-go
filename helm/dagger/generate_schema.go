package main

import (
	"context"
	"fmt"

	"github.com/creasty/defaults"
	"github.com/disaster37/dagger-library-go/helm/dagger/internal/dagger"
	"github.com/disaster37/dagger-library-go/helper"
	"github.com/gookit/validate"
)

type GenerateSchemaOption struct {
	Source     *dagger.Directory
	FileName   string `default:"values.schema.json"`
	ConfigFile string
	WithImage  string `default:"node:21-alpine"`
}

// GenerateSchema permit to generate helm schema
// It will return the values.schema.json file
func (m *Helm) GenerateSchema(
	ctx context.Context,

	// the source directory
	source *dagger.Directory,

	// the alternative image
	// +optional
	withImage string,
) (schemaFile *dagger.File, err error) {

	option := &GenerateSchemaOption{
		Source:    source,
		WithImage: withImage,
	}

	if err = defaults.Set(option); err != nil {
		return nil, err
	}

	if err = validate.Struct(option).ValidateErr(); err != nil {
		return nil, err
	}

	var container *dagger.Container
	if option.ConfigFile == "" {
		container = m.GetGeneratorContainer(ctx, option.Source, option.WithImage).
			WithExec(helper.ForgeCommand(fmt.Sprintf("readme-generator -s %s --values values.yaml", option.FileName)))
	} else {
		container = m.GetGeneratorContainer(ctx, option.Source, option.WithImage).
			WithExec(helper.ForgeCommand(fmt.Sprintf("readme-generator -c %s -s %s --values values.yaml", option.ConfigFile, option.FileName)))
	}

	schemaFile = container.File(option.FileName)

	return schemaFile, nil
}
