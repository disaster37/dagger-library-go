package image

import (
	"context"
	"fmt"
	"os"

	"dagger.io/dagger"
	"emperror.dev/errors"
	"github.com/disaster37/dagger-library-go/helper"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

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

// GetBuildCommand permit to get the command spec to add on cli
func GetBuildCommand(registryName, imageName string) *cli.Command {
	return &cli.Command{
		Name:  "buildImage",
		Usage: "Build the docker image",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "tag",
				Usage: "The image tag",
				Value: "staging",
			},
			&cli.BoolFlag{
				Name:  "push",
				Usage: "Push image on registry",
			},
			&cli.StringFlag{
				Name:     "registry-username",
				Usage:    "The username to connect on registry",
				Required: false,
				EnvVars:  []string{"REGISTRY_USERNAME"},
			},
			&cli.StringFlag{
				Name:     "registry-password",
				Usage:    "The password to connect on registry",
				Required: false,
				EnvVars:  []string{"REGISTRY_PASSWORD"},
			},
			&cli.StringFlag{
				Name:    "custom-ca-path",
				Usage:   "The custom ca full path file",
				EnvVars: []string{"CUSTOM_CA_PATH"},
			},
			&cli.StringFlag{
				Name:  "path",
				Usage: "The path context to build image",
				Value: ".",
			},
		},
		Action: func(c *cli.Context) (err error) {
			// initialize Dagger client
			client, err := helper.WithCustomCa(c.Context, c.String("custom-ca-path"), dagger.WithLogOutput(os.Stdout))
			if err != nil {
				panic(err)
			}
			defer client.Close()

			buildOption := &BuildImageOption{
				RegistryName:         registryName,
				Name:                 imageName,
				Tag:                  c.String("tag"),
				WithPush:             c.Bool("push"),
				WithRegistryUsername: c.String("registry-username"),
				WithRegistryPassword: c.String("registry-password"),
				PathContext:          c.String("path"),
			}

			return BuildImage(c.Context, client, buildOption)
		},
	}
}

// BuildImage permit to build image
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
		image := fmt.Sprintf("ghcr.io/hadolint/hadolint:%s", hadolint_version)

		_, err := client.
			Container().
			From(image).
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
		args = make([]dagger.BuildArg, 0)
		if os.Getenv("HTTP_PROXY") != "" {
			args = append(args, dagger.BuildArg{
				Name:  "HTTP_PROXY",
				Value: os.Getenv("HTTP_PROXY"),
			})
		}
		if os.Getenv("HTTPS_PROXY") != "" {
			args = append(args, dagger.BuildArg{
				Name:  "HTTPS_PROXY",
				Value: os.Getenv("HTTPS_PROXY"),
			})
		}
		if os.Getenv("NO_PROXY") != "" {
			args = append(args, dagger.BuildArg{
				Name:  "NO_PROXY",
				Value: os.Getenv("NO_PROXY"),
			})
		}
		if os.Getenv("http_proxy") != "" {
			args = append(args, dagger.BuildArg{
				Name:  "HTTP_PROXY",
				Value: os.Getenv("http_proxy"),
			})
		}
		if os.Getenv("https_proxy") != "" {
			args = append(args, dagger.BuildArg{
				Name:  "HTTPS_PROXY",
				Value: os.Getenv("https_proxy"),
			})
		}
		if os.Getenv("no_proxy") != "" {
			args = append(args, dagger.BuildArg{
				Name:  "NO_PROXY",
				Value: os.Getenv("no_proxy"),
			})
		}
	}

	// build using Dockerfile
	container := contextDir.DockerBuild(
		dagger.DirectoryDockerBuildOpts{
			BuildArgs: args,
		},
	)

	image := fmt.Sprintf("%s/%s:%s", option.RegistryName, option.Name, option.Tag)
	if option.WithPush {
		secret := client.SetSecret("password", option.WithRegistryPassword)

		ref, err := container.
			WithRegistryAuth(option.RegistryName, option.WithRegistryUsername, secret).
			Publish(
				ctx,
				image,
			)

		if err != nil {
			return errors.Wrapf(err, "Error when push image %s", image)
		}

		log.Infof("Published image to: %s", ref)
	} else {
		_, err = container.Export(ctx, "/dev/null")
		if err != nil {
			return errors.Wrapf(err, "Error when build image %s", image)
		}
	}

	return nil
}
