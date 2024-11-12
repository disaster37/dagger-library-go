// A generated module for Git functions
//
// This module has been generated via dagger init and serves as a reference to
// basic module structure as you get started with Dagger.
//
// Two functions have been pre-created. You can modify, delete, or add to them,
// as needed. They demonstrate usage of arguments and return types using simple
// echo and grep commands. The functions can be called from the dagger CLI or
// from one of the SDKs.
//
// The first line in this comment block is a short description line and the
// rest is a long description with more detail on the module's purpose or usage,
// if appropriate. All modules should have a short description.

package main

import (
	"context"
	"dagger/git/internal/dagger"
	"fmt"

	"dagger.io/dagger/dag"
	"emperror.dev/errors"
	"github.com/disaster37/dagger-library-go/lib/helper"
)

type Git struct {
	BaseContainer *dagger.Container
}

// Default contsructor
func New(
	// base container
	// +optional
	baseContainer *dagger.Container,
) *Git {
	git := &Git{}
	if baseContainer != nil {
		git.BaseContainer = baseContainer
	} else {
		git.BaseContainer = git.GetBaseContainer()
	}

	return git
}

// BaseContainer permit to get the base container
func (m *Git) GetBaseContainer() *dagger.Container {
	return dag.Container().
		From("alpine:latest").
		WithExec(helper.ForgeCommand("apk add --update git")).
		WithExec(helper.ForgeCommand("git config --global --add --bool push.autoSetupRemote true")).
		WithExec(helper.ForgeCommand("git config --global --add safe.directory /project"))
}

// SetConfig permit to set git config
func (m *Git) SetConfig(
	ctx context.Context,

	// The git username
	username string,

	// The git email
	email string,

	// The git base repo URL
	// +optional
	// +default="github.com"
	baseRepoUrl string,

	// The git token
	// +optional
	token *dagger.Secret,
) (*Git, error) {
	m.BaseContainer = m.BaseContainer.
		WithExec(helper.ForgeCommandf("git config --global user.name %s", username)).
		WithExec(helper.ForgeCommandf("git config --global user.email %s", email))

	if token != nil {
		tokenPlain, err := token.Plaintext(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "Error when get token")
		}
		m.BaseContainer = m.BaseContainer.
			WithNewFile("/.git-credentials", fmt.Sprintf(`https://%s:%s@%s`, username, tokenPlain, baseRepoUrl)).
			WithExec([]string{"git", "config", "--global", "credential.helper", "store --file /.git-credentials"})
	}
	return m, nil
}

// SetRepo permit to set git repo
func (m *Git) SetRepo(
	ctx context.Context,

	// the source directory
	source *dagger.Directory,

	// The git email
	// +default="main"
	branch string,

) (*Git, error) {

	m.BaseContainer = m.BaseContainer.
		WithDirectory("/project", source).
		WithWorkdir("/project").
		WithExec(helper.ForgeScript(`
git name-rev --name-only HEAD | grep tags
RETCODE=$?
if [ $RETCODE -eq 0 ]; then
	git fetch origin %s:%s
	git checkout %s
fi
		`, branch, branch, branch))
	return m, nil
}

// CommitAndPush permit to commit and push
func (m *Git) CommitAndPush(
	ctx context.Context,

	// The commit message
	message string,
) (string, error) {

	return m.BaseContainer.
		WithExec(helper.ForgeCommand("git add -A")).
		WithExec(helper.ForgeScript(
			`
if [ -n "\$(git status --untracked-files=no --porcelain)" ]; then
	git commit -m "%s"
	git push
fi
		`, message)).
		Stdout(ctx)
}
