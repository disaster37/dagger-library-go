package git

import (
	"context"
	"fmt"

	"dagger.io/dagger"
	"github.com/creasty/defaults"
	"github.com/gookit/validate"
	"github.com/urfave/cli/v2"
)

type GitOption struct {
	PathContext string `default:"."`
	Author      string `validate:"required"`
	Email       string `validate:"required"`
	Token       string `validate:"required"`
	WithProxy   bool   `default:"true"`
}

func InitGitFlag(app *cli.App) {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:     "git-token",
			Usage:    "The git token",
			Required: false,
			EnvVars:  []string{"GIT_TOKEN"},
		},
	}

	app.Flags = append(app.Flags, flags...)
}

func CommitAndPush(ctx context.Context, client *dagger.Client, option *GitOption) (err error) {
	if err = defaults.Set(option); err != nil {
		panic(err)
	}

	if err = validate.Struct(option).ValidateErr(); err != nil {
		panic(err)
	}

	_, err = getGitContainer(client, option.PathContext, option.WithProxy).
		WithEntrypoint([]string{"/bin/sh", "-c"}).
		WithExec([]string{fmt.Sprintf("git config --global user.email %s", option.Email)}).
		WithExec([]string{fmt.Sprintf("git config --global user.name %s", option.Author)}).
		WithExec([]string{fmt.Sprintf("git remote set-url origin https://%s@$(git config remote.origin.url)", option.Token)}).
		WithExec([]string{"git checkout ${git branch --no-color --show-current}"}).
		WithExec([]string{"git add -A "}).
		WithExec([]string{"git commit -m \"push back from pipeline\""}).
		WithExec([]string{"git push"}).
		Stdout(ctx)

	return err

}
