package main

import (
	"context"

	"dagger/helm/internal/dagger"

	"emperror.dev/errors"
	"github.com/disaster37/dagger-library-go/lib/v2/helper"
	"gopkg.in/yaml.v3"
)

// Push helm chart on registry (OCI format only)
// It will return the updated Chart.yaml file with the expected version
func (m *Helm) Push(
	ctx context.Context,

	// The registry url
	registryUrl string,

	// The repository name
	repositoryName string,

	// The version
	version string,

) (chartFile *dagger.File, err error) {

	// Update the chart version
	chartFile = m.UpdateChart(
		ctx,
		".version",
		version,
	)
	helmModule := m.WithSource(m.Src.WithFile("Chart.yaml", chartFile))

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
	ociUrl := "oci://" + registryUrl + "/" + repositoryName + "/" + chartName + ":" + version

	_, err = helmModule.HelmContainer.
		WithExec(helper.ForgeCommand("helm dependency update")).
		WithExec(helper.ForgeCommand("helm package -u .")).
		WithExec(helper.ForgeCommandf("helm push %s-%s.tgz oci://%s/%s", chartName, version, registryUrl, repositoryName)).
		WithExec(helper.ForgeCommandf("echo 'OCI URL: %s'", ociUrl)).
		Stdout(ctx)

	if err != nil {
		return nil, errors.Wrap(err, "Error when package and push helm chart")
	}

	return chartFile, nil
}
