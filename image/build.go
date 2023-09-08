package image

import (
	"context"
	"fmt"
	"os"

	"dagger.io/dagger"
	"emperror.dev/errors"
	"github.com/disaster37/dagger-library-go/helper"
	log "github.com/sirupsen/logrus"

	"github.com/creasty/defaults"
	"github.com/gookit/validate"
)

type BuildImageOption struct {
	WithLint             bool   `default:"true"`
	WithProxy            bool   `default:"true"`
	WithPush             bool   `default:"false"`
	WithRegistryUsername string `validate:"validateRegistryAuth"`
	WithRegistryPassword string `validate:"validateRegistryAuth"`
	RegistryName         string `validate:"required"`
	Name                 string `validate:"required"`
	Tag                  string `default:"latest"`
	PathContext          string `default:"."`
}

func (h BuildImageOption) ValidateRegistryAuth(val string) bool {
	if h.WithPush && val == "" {
		return false
	}

	return true
}

func BuildImage(ctx context.Context, client *dagger.Client, option *BuildImageOption) (err error) {

	if err = defaults.Set(option); err != nil {
		panic(err)
	}

	if err = validate.Struct(option).ValidateErr(); err != nil {
		panic(err)
	}

	// get build context directory
	contextDir := client.Host().Directory(option.PathContext)

	// Lint image if needed
	if option.WithLint {
		_, err := client.
			Container().
			From("ghcr.io/hadolint/hadolint:latest-alpine").
			WithDirectory("/project", client.Host().Directory(option.PathContext)).
			WithWorkdir("/project").
			WithExec(helper.ForgeCommand("/bin/hadolint --failure-threshold error Dockerfile")).
			Stdout(ctx)

		if err != nil {
			return errors.Wrap(err, "Error when lint Dockerfile")
		}
	}

	// Compute build args
	var args []dagger.BuildArg
	if option.WithProxy {
		args = []dagger.BuildArg{
			{
				Name:  "HTTP_PROXY",
				Value: os.Getenv("HTTP_PROXY"),
			},
			{
				Name:  "HTTPS_PROXY",
				Value: os.Getenv("HTTPS_PROXY"),
			},
			{
				Name:  "NO_PROXY",
				Value: os.Getenv("NO_PROXY"),
			},
		}
	}

	// build using Dockerfile
	container := client.
		Container().
		Build(
			contextDir,
			dagger.ContainerBuildOpts{
				BuildArgs: args,
			},
		)

	if option.WithPush {
		secret := client.SetSecret("password", option.WithRegistryPassword)
		ref, err := container.
			WithRegistryAuth(option.RegistryName, option.WithRegistryUsername, secret).
			Publish(
				ctx,
				fmt.Sprintf("%s:%s", option.Name, option.Tag),
			)

		if err != nil {
			return errors.Wrapf(err, "Error when push image %s:%s", option.Name, option.Tag)
		}

		log.Infof("Published image to :%s", ref)
	} else {
		_, err = container.Stdout(ctx)
		if err != nil {
			return errors.Wrapf(err, "Error when build image %s:%s", option.Name, option.Tag)
		}
	}

	return nil
}
