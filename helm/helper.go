package helm

import (
	"fmt"

	"dagger.io/dagger"
	"github.com/disaster37/dagger-library-go/helper"
)

const (
	yq_version string = "4.35.2"
)

func getHelmContainer(client *dagger.Client, withImage string, path string, withProxy bool) *dagger.Container {

	container := client.
		Container().
		From(withImage)

	if withProxy {
		container = helper.WithProxy(container)
	}

	return container.
		WithDirectory("/project", client.Host().Directory(path, dagger.HostDirectoryOpts{Exclude: []string{"ci"}})).
		WithWorkdir("/project")
}

func getGeneratorContainer(client *dagger.Client, withImage string, path string, withProxy bool) *dagger.Container {

	container := client.
		Container().
		From(withImage)

	if withProxy {
		container = helper.WithProxy(container)
	}

	return container.
		WithDirectory("/project", client.Host().Directory(path, dagger.HostDirectoryOpts{Exclude: []string{"ci"}})).
		WithWorkdir("/project").
		WithExec(helper.ForgeCommand("npm install -g @bitnami/readme-generator-for-helm"))
}

func getYQContainer(client *dagger.Client, path string) *dagger.Container {
	image := fmt.Sprintf("mikefarah/yq:%s", yq_version)

	return client.
		Container().
		From(image).
		WithDirectory("/project", client.Host().Directory(path, dagger.HostDirectoryOpts{Exclude: []string{"ci"}})).
		WithWorkdir("/project")
}
