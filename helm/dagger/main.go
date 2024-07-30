// A generated module for Helm functions
//
// This module has been generated via dagger init and serves as a reference to
// basic module structure as you get started with Dagger.
//
// Two functions have been pre-created. You can modify, delete, or add to them,
// as needed. They demonstrate usage of arguments and return types using simple
// echo and grep commands. The functions can be called from the dagger CLI or
// from one of the SDKs.
//
// The first line in this comment block is a short description line and the
// rest is a long description with more detail on the module's purpose or usage,
// if appropriate. All modules should have a short description.

package main

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/creasty/defaults"
	"github.com/disaster37/dagger-library-go/helm/dagger/internal/dagger"
	"github.com/disaster37/dagger-library-go/helper"
	"github.com/gookit/validate"
)

const (
	defaultGeneratorImage string = "node:21-alpine"
)

type Helm struct{}

type GenerateSchemaOption struct {
	Source     *dagger.Directory
	FileName   string `default:"values.schema.json"`
	ConfigFile string
	WithImage  string `default:"node:21-alpine"`
}

type GenerateDocumentationOption struct {
	Source     *dagger.Directory
	FileName   string `default:"README.md"`
	ConfigFile string
	WithImage  string `default:"node:21-alpine"`
}

type LintOption struct {
	Source    *dagger.Directory
	WithImage string `default:"alpine/helm:3.14.3"`
}

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
	if _, err = schemaFile.Export(ctx, fmt.Sprintf("%s/%s", option.Source, option.FileName)); err != nil {
		return nil, errors.Wrap(err, "Error when generate helm schema")
	}

	return schemaFile, nil
}

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
	if _, err = readmeFile.Export(ctx, fmt.Sprintf("%s/%s", option.Source, option.FileName)); err != nil {
		return nil, errors.Wrap(err, "Error when generate helm readme")
	}

	return readmeFile, nil

}

func (m *Helm) Lint(
	ctx context.Context,

	// the source directory
	source *dagger.Directory,

	// the alternative image
	// +optional
	withImage *string,

	// Files to inject on containers
	// +optional
	files ...*dagger.File,
) (stdout string, err error) {

	option := &LintOption{
		Source: source,
	}
	if withImage != nil {
		option.WithImage = *withImage
	}

	if err = defaults.Set(option); err != nil {
		panic(err)
	}

	if err = validate.Struct(option).ValidateErr(); err != nil {
		panic(err)
	}

	container := m.GetHelmContainer(ctx, option)
	for _, file := range files {
		fileName, err := file.Name(ctx)
		if err != nil {
			return "", err
		}
		container = container.WithFile(fileName, file)
	}

	return container.
		WithExec(helper.ForgeCommand("helm dependency update")).
		WithExec(helper.ForgeCommand("helm lint .")).
		Stdout(ctx)
}

func (m *Helm) GetGeneratorContainer(ctx context.Context, source *dagger.Directory, withImage string) *dagger.Container {
	return dag.Container().
		From(withImage).
		WithDirectory("/project", source).
		WithWorkdir("/project").
		WithExec(helper.ForgeCommand("npm install -g @bitnami/readme-generator-for-helm"))
}

func (m *Helm) GetHelmContainer(ctx context.Context, params *LintOption) *dagger.Container {
	return dag.Container().
		From(params.WithImage).
		WithDirectory("/project", params.Source).
		WithWorkdir("/project")
}
