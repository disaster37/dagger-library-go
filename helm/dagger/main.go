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
	WithProxy  bool   `default:"true"`
	FileName   string `default:"values.schema.json"`
	ConfigFile string
	WithImage  string `default:"node:21-alpine"`
}

type LintOption struct {
	Source    *dagger.Directory
	WithProxy bool   `default:"true"`
	WithImage string `default:"alpine/helm:3.14.3"`
}

func (m *Helm) GenerateSchema(ctx context.Context, source *dagger.Directory, withImage *string, withProxy *bool) (schemaFile *dagger.File, err error) {

	option := &GenerateSchemaOption{
		Source: source,
	}
	if withImage != nil {
		option.WithImage = *withImage
	}
	if withProxy != nil {
		option.WithProxy = *withProxy
	}

	if err = defaults.Set(option); err != nil {
		return nil, err
	}

	if err = validate.Struct(option).ValidateErr(); err != nil {
		return nil, err
	}

	var container *dagger.Container
	if option.ConfigFile == "" {
		container = m.GetGeneratorContainer(ctx, option).
			WithExec(helper.ForgeCommand(fmt.Sprintf("readme-generator -s %s --values values.yaml", option.FileName)))
	} else {
		container = m.GetGeneratorContainer(ctx, option).
			WithExec(helper.ForgeCommand(fmt.Sprintf("readme-generator -c %s -s %s --values values.yaml", option.ConfigFile, option.FileName)))
	}

	schemaFile = container.File(option.FileName)

	return schemaFile, nil
}

func (m *Helm) GenerateDocumentation(ctx context.Context, source *dagger.Directory, withImage *string, withProxy *bool) (readmeFile *dagger.File, err error) {
	option := &GenerateSchemaOption{
		Source: source,
	}
	if withImage != nil {
		option.WithImage = *withImage
	}
	if withProxy != nil {
		option.WithProxy = *withProxy
	}

	if err = defaults.Set(option); err != nil {
		return nil, err
	}

	if err = validate.Struct(option).ValidateErr(); err != nil {
		return nil, err
	}

	var container *dagger.Container
	if option.ConfigFile == "" {
		container = m.GetGeneratorContainer(ctx, option).
			WithExec(helper.ForgeCommand(fmt.Sprintf("readme-generator -r %s --values values.yaml", option.FileName)))
	} else {
		container = m.GetGeneratorContainer(ctx, option).
			WithExec(helper.ForgeCommand(fmt.Sprintf("readme-generator -c %s -r %s --values values.yaml", option.ConfigFile, option.FileName)))
	}

	readmeFile = container.File(option.FileName)

	return readmeFile, nil

}

func (m *Helm) Lint(ctx context.Context, source *dagger.Directory, withImage *string, withProxy *bool, files ...*dagger.File) (stdout string, err error) {

	option := &LintOption{
		Source: source,
	}
	if withImage != nil {
		option.WithImage = *withImage
	}
	if withProxy != nil {
		option.WithProxy = *withProxy
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

func (m *Helm) GetGeneratorContainer(ctx context.Context, params *GenerateSchemaOption) *dagger.Container {
	return dag.Container().
		From(params.WithImage).
		WithDirectory("/project", params.Source).
		WithWorkdir("/project").
		WithExec(helper.ForgeCommand("npm install -g @bitnami/readme-generator-for-helm"))
}

func (m *Helm) GetHelmContainer(ctx context.Context, params *LintOption) *dagger.Container {
	return dag.Container().
		From(params.WithImage).
		WithDirectory("/project", params.Source).
		WithWorkdir("/project")
}
