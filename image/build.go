package main

import (
	"context"
	"dagger/image/internal/dagger"
	"fmt"

	"emperror.dev/errors"
)

type ImageBuild struct {
	// +private
	Container *dagger.Container
}

// GetContainer permit to get the container
func (m *ImageBuild) GetContainer() *dagger.Container {
	return m.Container
}

// Push permit to push image
func (m *ImageBuild) Push(
	ctx context.Context,

	// The repository name
	repositoryName string,

	// The version
	version string,

	// The registry username
	// +optional
	withRegistryUsername *dagger.Secret,

	// The registry password
	// +optional
	withRegistryPassword *dagger.Secret,

	// The registry url
	registryUrl string,
) (string, error) {

	if withRegistryUsername != nil && withRegistryPassword != nil {
		username, err := withRegistryUsername.Plaintext(ctx)
		if err != nil {
			return "", errors.Wrap(err, "Error when get registry username")
		}
		m.Container = m.Container.WithRegistryAuth(registryUrl, username, withRegistryPassword)
	}

	return m.Container.
		Publish(
			ctx,
			fmt.Sprintf("%s/%s:%s", registryUrl, repositoryName, version),
		)
}
