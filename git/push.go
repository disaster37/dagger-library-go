package git

import (
	"context"
	"fmt"

	"dagger.io/dagger"
	"github.com/creasty/defaults"
	"github.com/gookit/validate"
)

type GitOption struct {
	PathContext string `default:"."`
	Author      string `validate:"required"`
	Email       string `validate:"required"`
}

func CommitAndPush(ctx context.Context, client *dagger.Client, option *GitOption) (err error) {
	if err = defaults.Set(option); err != nil {
		panic(err)
	}

	if err = validate.Struct(option).ValidateErr(); err != nil {
		panic(err)
	}

	_, err = getGitContainer(client, option.PathContext).
		WithEntrypoint([]string{"/bin/sh", "-c"}).
		WithExec([]string{fmt.Sprintf("git config --global user.email %s", option.Email)}).
		WithExec([]string{fmt.Sprintf("git config --global user.name %s", option.Author)}).
		WithExec([]string{"git add -A "}).
		WithExec([]string{"git commit -m \"push back from pipeline\""}).
		WithExec([]string{"git push"}).
		Stdout(ctx)

	return err

}
