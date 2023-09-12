package helper

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"emperror.dev/errors"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

// WithCustomCa permit to inject CA certificat on dagger-engine if manager by dagger cli
func WithCustomCa(ctx context.Context, caPath string) (err error) {

	if os.Getenv("_DAGGER_RUNNER_HOST") != "" {
		return nil
	}

	f, err := os.Open(caPath)
	if err != nil {
		return errors.Wrapf(err, "Error when open file %s", caPath)
	}
	defer f.Close()

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: filters.NewArgs(filters.KeyValuePair{
		Key:   "name",
		Value: "dagger-engine",
	})})

	if err != nil {
		return errors.Wrap(err, "Error when list containers")
	}

	for _, container := range containers {
		cli.CopyToContainer(ctx, container.ID, fmt.Sprintf("/etc/ssl/certs/%s", filepath.Base(caPath)), f, types.CopyToContainerOptions{})
	}

	return nil

}
