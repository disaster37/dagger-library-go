package main

import (
	"context"
	"fmt"

	"dagger/helm/internal/dagger"

	"emperror.dev/errors"
	"github.com/creasty/defaults"
	"github.com/disaster37/dagger-library-go/lib/helper"
	"github.com/gookit/validate"
	"gopkg.in/yaml.v3"
)

type PushOption struct {
	Source         *dagger.Directory `validate:"required"`
	RegistryUrl    string            `validate:"required"`
	RepositoryName string            `validate:"required"`
	Version        string            `validate:"required"`
	WithFiles      []*dagger.File
}

// Push helm chart on registry
// It will return the updated Chart.yaml file with the expected version
func (m *Helm) Push(
	ctx context.Context,

	// the source directory
	source *dagger.Directory,

	// The registry url
	registryUrl string,

	// The repository name
	repositoryName string,

	// The version
	version string,

	// Files to inject on containers
	// +optional
	withFiles []*dagger.File,
) (chartFile *dagger.File, err error) {

	option := &PushOption{
		Source:         source,
		RegistryUrl:    registryUrl,
		RepositoryName: repositoryName,
		Version:        version,
		WithFiles:      withFiles,
	}

	if err = defaults.Set(option); err != nil {
		return nil, err
	}

	if err = validate.Struct(option).ValidateErr(); err != nil {
		return nil, err
	}

	// Update the chart version
	chartFile = m.BaseYqContainer.
		WithDirectory("/project", source).
		WithWorkdir("/project").
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

	_, err = m.BaseHelmContainer.
		WithDirectory("/project", source).
		WithWorkdir("/project").
		WithFile("Chart.yaml", chartFile).
		WithFiles("/project", option.WithFiles).
		WithExec(helper.ForgeCommand("helm dependency update")).
		WithExec(helper.ForgeCommand("helm package -u .")).
		WithExec(helper.ForgeCommandf("helm push %s-%s.tgz oci://%s/%s", chartName, option.Version, option.RegistryUrl, option.RepositoryName)).
		Stdout(ctx)

	if err != nil {
		return nil, errors.Wrap(err, "Error when package and push helm chart")
	}

	return chartFile, nil

}
