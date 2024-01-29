package image

import (
	"fmt"

	"dagger.io/dagger"
)

const (
	hadolint_version string = "2.12.0"
)

func getHadolintContainer(client *dagger.Client, path string) *dagger.Container {
	image := fmt.Sprintf("ghcr.io/hadolint/hadolint:%s", hadolint_version)

	return client.
		Container().
		From(image).
		WithDirectory("/project", client.Host().Directory(path, dagger.HostDirectoryOpts{Exclude: []string{"ci"}})).
		WithWorkdir("/project")
}
