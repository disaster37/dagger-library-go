package helm

import (
	"context"
	"fmt"

	"dagger.io/dagger"
	"emperror.dev/errors"
	"github.com/creasty/defaults"
	"github.com/disaster37/dagger-library-go/helper"
	"github.com/gookit/validate"
)

type GenerateDocumentationOption struct {
	WithProxy   bool   `default:"true"`
	PathContext string `default:"."`
	FileName    string `default:"README.md"`
	ConfigFile  string
	WithImage   string `default:"node:21-alpine"`
}

// GenerateDocumentation permit to generate helm documentation
func GenerateDocumentation(ctx context.Context, client *dagger.Client, option *GenerateDocumentationOption) (files map[string]*dagger.File, err error) {

	if err = defaults.Set(option); err != nil {
		panic(err)
	}

	if err = validate.Struct(option).ValidateErr(); err != nil {
		panic(err)
	}

	var container *dagger.Container
	if option.ConfigFile == "" {
		container = getGeneratorContainer(client, option.WithImage, option.PathContext, option.WithProxy).
			WithExec(helper.ForgeCommand(fmt.Sprintf("readme-generator -r %s --values values.yaml", option.FileName)))
	} else {
		container = getGeneratorContainer(client, option.WithImage, option.PathContext, option.WithProxy).
			WithExec(helper.ForgeCommand(fmt.Sprintf("readme-generator -c %s -r %s --values values.yaml", option.ConfigFile, option.FileName)))
	}

	readmeFile := container.File(option.FileName)
	if _, err = readmeFile.Export(ctx, fmt.Sprintf("%s/%s", option.PathContext, option.FileName)); err != nil {
		return nil, errors.Wrap(err, "Error when generate helm readme")
	}

	files = map[string]*dagger.File{
		option.FileName: readmeFile,
	}

	return files, nil
}
