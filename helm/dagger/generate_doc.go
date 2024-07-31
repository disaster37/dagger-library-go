package main

import (
	"context"
	"fmt"

	"github.com/creasty/defaults"
	"github.com/disaster37/dagger-library-go/helm/dagger/internal/dagger"
	"github.com/disaster37/dagger-library-go/helper"
	"github.com/gookit/validate"
)

type GenerateDocumentationOption struct {
	Source     *dagger.Directory
	FileName   string `default:"README.md"`
	ConfigFile string
	WithImage  string `default:"node:21-alpine"`
}

// GenerateDocumentation permit to generate helm documentation
// It will return the readme file
func (m *Helm) GenerateDocumentation(
	ctx context.Context,

	// the source directory
	source *dagger.Directory,

	// the alternative image
	// +optional
	withImage string,
) (readmeFile *dagger.File, err error) {
	option := &GenerateDocumentationOption{
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
			WithExec(helper.ForgeCommand(fmt.Sprintf("readme-generator -r %s --values values.yaml", option.FileName)))
	} else {
		container = m.GetGeneratorContainer(ctx, option.Source, option.WithImage).
			WithExec(helper.ForgeCommand(fmt.Sprintf("readme-generator -c %s -r %s --values values.yaml", option.ConfigFile, option.FileName)))
	}

	readmeFile = container.File(option.FileName)

	return readmeFile, nil

}
