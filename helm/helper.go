package helm

import (
	"fmt"

	"dagger.io/dagger"
)

func getHelmContainer(client *dagger.Client, path string) *dagger.Container {
	image := fmt.Sprintf("alpine/helm:%s", helm_version)

	return client.
		Container().
		From(image).
		WithDirectory("/project", client.Host().Directory(path)).
		WithWorkdir("/project")
}
