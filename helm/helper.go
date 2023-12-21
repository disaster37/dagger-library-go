package helm

import (
	"fmt"

	"dagger.io/dagger"
	"github.com/disaster37/dagger-library-go/helper"
)

func getHelmContainer(client *dagger.Client, path string) *dagger.Container {
	image := fmt.Sprintf("alpine/helm:%s", helm_version)

	return client.
		Container().
		From(image).
		WithDirectory("/project", client.Host().Directory(path, dagger.HostDirectoryOpts{Exclude: []string{"ci"}})).
		WithWorkdir("/project")
}

func getGeneratorContainer(client *dagger.Client, path string, withProxy bool) *dagger.Container {

	image := fmt.Sprintf("node:%s", node_version)

	container := client.
		Container().
		From(image)

	if withProxy {
		container = helper.WithProxy(container)
	}

	return container.
		WithDirectory("/project", client.Host().Directory(path, dagger.HostDirectoryOpts{Exclude: []string{"ci"}})).
		WithWorkdir("/project").
		WithExec(helper.ForgeCommand("npm install -g @bitnami/readme-generator-for-helm"))
}
