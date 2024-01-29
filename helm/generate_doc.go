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
}

// BuildHelm permit to build helm chart
func GenerateDocumentation(ctx context.Context, client *dagger.Client, option *GenerateDocumentationOption) (err error) {

	if err = defaults.Set(option); err != nil {
		panic(err)
	}

	if err = validate.Struct(option).ValidateErr(); err != nil {
		panic(err)
	}

	_, err = getGeneratorContainer(client, option.PathContext, option.WithProxy).
		WithExec(helper.ForgeCommand(fmt.Sprintf("readme-generator -r %s --values values.yaml", option.FileName))).
		File(option.FileName).
		Export(ctx, fmt.Sprintf("%s/%s", option.PathContext, option.FileName))

	if err != nil {
		return errors.Wrap(err, "Error when generate helm readme")
	}

	return nil
}
