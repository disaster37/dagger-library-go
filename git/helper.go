package git

import (
	"fmt"

	"dagger.io/dagger"
	"github.com/disaster37/dagger-library-go/helper"
)

const (
	git_version string = "2.43.0"
)

func getGitContainer(client *dagger.Client, path string, withProxy bool) *dagger.Container {
	image := fmt.Sprintf("bitnami/git:%s", git_version)

	container := client.
		Container().
		From(image).
		WithDirectory("/project", client.Host().Directory(path, dagger.HostDirectoryOpts{Exclude: []string{"ci"}})).
		WithWorkdir("/project")

	if withProxy {
		container = helper.WithProxy(container)
	}

	return container
}
