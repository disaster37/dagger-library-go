package main

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/creasty/defaults"
	"github.com/disaster37/dagger-library-go/helm/dagger/internal/dagger"
	"github.com/disaster37/dagger-library-go/helper"
	"github.com/gookit/validate"
	"gopkg.in/yaml.v3"
)

type PushOption struct {
	Source               *dagger.Directory
	WithRegistryUsername *dagger.Secret `validate:"required"`
	WithRegistryPassword *dagger.Secret `validate:"required"`
	RegistryUrl          string         `validate:"required"`
	RepositoryName       string         `validate:"required"`
	Version              string         `validate:"required"`
	WithFiles            []*dagger.File
	WithImage            string `default:"alpine/helm:3.14.3"`
	WithYQImage          string `default:"mikefarah/yq:4.35.2"`
}

// Push helm chart on registry
// It will return the updated Chart.yaml file with the expected version
func (m *Helm) Push(
	ctx context.Context,

	// the source directory
	source *dagger.Directory,

	// The registry username
	withRegistryUsername *dagger.Secret,

	// The registry password
	withRegistryPassword *dagger.Secret,

	// The registry url
	registryUrl string,

	// The repository name
	repositoryName string,

	// The version
	version string,

	// Files to inject on containers
	// +optional
	withFiles []*dagger.File,

	// The alternative helm image
	// +optional
	withImage string,

	// The alternative YQ image
	// +optional
	withYQImage string,

	// The registry username
) (chartFile *dagger.File, err error) {

	option := &PushOption{
		Source:               source,
		WithRegistryUsername: withRegistryUsername,
		WithRegistryPassword: withRegistryPassword,
		RegistryUrl:          registryUrl,
		RepositoryName:       repositoryName,
		Version:              version,
		WithFiles:            withFiles,
		WithImage:            withImage,
		WithYQImage:          withYQImage,
	}

	if err = defaults.Set(option); err != nil {
		return nil, err
	}

	if err = validate.Struct(option).ValidateErr(); err != nil {
		return nil, err
	}

	// Update the chart version
	chartFile = m.GetYqContainer(ctx, option.Source, option.WithYQImage).
		WithExec(
			[]string{"yq", "--inplace", fmt.Sprintf(".version = \"%s\"", option.Version), "Chart.yaml"},
			dagger.ContainerWithExecOpts{InsecureRootCapabilities: true},
		).
		File("Chart.yaml")

	chartContends, err := chartFile.Contents(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Error when read chart file")
	}

	// Read chart file to get the chart name
	dataChart := make(map[string]any)
	if err = yaml.Unmarshal([]byte(chartContends), &dataChart); err != nil {
		return nil, errors.Wrap(err, "Error when decode YAML file")
	}
	chartName := dataChart["name"].(string)

	// Package and push

	_, err = m.GetHelmContainer(ctx, option.Source, option.WithImage).
		WithFile("Chart.yaml", chartFile).
		WithFiles("/project", option.WithFiles).
		WithSecretVariable("REGISTRY_USERNAME", withRegistryUsername).
		WithSecretVariable("REGISTRY_PASSWORD", withRegistryPassword).
		WithExec(helper.ForgeCommand("helm dependency update")).
		WithExec(helper.ForgeCommand("helm package -u .")).
		WithExec([]string{"sh", "-c", fmt.Sprintf("helm registry login -u $REGISTRY_USERNAME -p $REGISTRY_PASSWORD %s", option.RegistryUrl)}).
		WithExec(helper.ForgeCommand(fmt.Sprintf("helm push %s-%s.tgz oci://%s/%s", chartName, option.Version, option.RegistryUrl, option.RepositoryName))).
		Stdout(ctx)

	if err != nil {
		return nil, errors.Wrap(err, "Error when package and push helm chart")
	}

	return chartFile, nil

}
