package helm

import (
	"dagger.io/dagger"
	"github.com/disaster37/dagger-library-go/helper"
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

func getYQContainer(client *dagger.Client, withImage string, path string) *dagger.Container {
	return client.
		Container().
		From(withImage).
		WithDirectory("/project", client.Host().Directory(path, dagger.HostDirectoryOpts{Exclude: []string{"ci"}})).
		WithWorkdir("/project")
}
