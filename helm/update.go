package main

import (
	"context"
	"dagger/helm/internal/dagger"
	"fmt"
)

// Update Chart.yaml file
func (m *Helm) UpdateChart(
	ctx context.Context,

	// the source directory
	source *dagger.Directory,

	// The key to update on yaml file
	key string,

	// The new value to put
	value string,

) (chartFile *dagger.File) {

	chartFile = m.BaseYqContainer.
		WithDirectory("/project", source).
		WithWorkdir("/project").
		WithExec(
			[]string{"yq", "--inplace", fmt.Sprintf("%s = \"%s\"", key, value), "Chart.yaml"},
			dagger.ContainerWithExecOpts{InsecureRootCapabilities: true},
		).
		File("Chart.yaml")

	return chartFile
}

// Update values.yaml file
func (m *Helm) UpdateValues(
	ctx context.Context,

	// the source directory
	source *dagger.Directory,

	// The key to update on yaml file
	key string,

	// The new value to put
	value string,

) (valueFile *dagger.File) {

	valueFile = m.BaseYqContainer.
		WithDirectory("/project", source).
		WithWorkdir("/project").
		WithExec(
			[]string{"yq", "--inplace", fmt.Sprintf("%s = \"%s\"", key, value), "values.yaml"},
			dagger.ContainerWithExecOpts{InsecureRootCapabilities: true},
		).
		File("values.yaml")

	return valueFile
}
