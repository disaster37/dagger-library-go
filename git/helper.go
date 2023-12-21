package git

import (
	"fmt"

	"dagger.io/dagger"
)

const (
	git_version string = "2.43.0"
)

func getGitContainer(client *dagger.Client, path string) *dagger.Container {
	image := fmt.Sprintf("bitnami/git:%s", git_version)

	return client.
		Container().
		From(image).
		WithDirectory("/project", client.Host().Directory(path, dagger.HostDirectoryOpts{Exclude: []string{"ci"}})).
		WithWorkdir("/project")
}
