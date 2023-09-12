package helper

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"emperror.dev/errors"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/sirupsen/logrus"
)

// WithCustomCa permit to inject CA certificat on dagger-engine if manager by dagger cli
func WithCustomCa(ctx context.Context, caPath string) (err error) {

	if os.Getenv("_DAGGER_RUNNER_HOST") != "" {
		return nil
	}

	// Read ca file
	f, err := os.Open(caPath)
	if err != nil {
		return errors.Wrapf(err, "Error when open file %s", caPath)
	}
	defer f.Close()

	// Prepare archive to upload it on container
	dstPath := fmt.Sprintf("/etc/ssl/certs/%s", filepath.Base(caPath))
	dstInfo := archive.CopyInfo{Path: dstPath}
	srcInfo, err := archive.CopyInfoSourcePath(caPath, true)
	if err != nil {
		return err
	}
	srcArchive, err := archive.TarResource(srcInfo)
	if err != nil {
		return err
	}
	defer srcArchive.Close()
	dstDir, preparedArchive, err := archive.PrepareArchiveCopy(srcArchive, srcInfo, dstInfo)
	if err != nil {
		return err
	}
	defer preparedArchive.Close()

	// Open docker connection
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: filters.NewArgs(filters.KeyValuePair{
		Key:   "name",
		Value: "dagger-engine",
	})})

	if err != nil {
		return errors.Wrap(err, "Error when list containers")
	}

	for _, c := range containers {

		// Check if file already exist on container
		_, err := cli.ContainerStatPath(ctx, c.ID, dstPath)
		if err != nil {
			if client.IsErrNotFound(err) {
				logrus.Infof("File %s already exist on container %s", dstPath, c.ID)
				continue
			}
			return errors.Wrapf(err, "Error when stats certificat on container %s", c.ID)
		}

		logrus.Infof("Inject %s on container %s", caPath, c.ID)
		if err = cli.CopyToContainer(ctx, c.ID, dstDir, preparedArchive, types.CopyToContainerOptions{AllowOverwriteDirWithFile: false, CopyUIDGID: false}); err != nil {
			return errors.Wrapf(err, "Error when inject %s on container %s", caPath, c.ID)
		}
		if err = cli.ContainerRestart(ctx, c.ID, container.StopOptions{}); err != nil {
			return errors.Wrapf(err, "Error whe restart container %s", c.ID)
		}
	}

	return nil

}
