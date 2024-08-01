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
)

type Git struct {
	baseContainer *dagger.Container
}

// Default contsructor
func New(
	// base container
	// +optional
	baseContainer *dagger.Container,
) *Git {
	git := &Git{}
	if baseContainer != nil {
		git.baseContainer = baseContainer
	} else {
		git.baseContainer = git.BaseContainer()
	}

	return git
}

// BaseContainer permit to get the base container
func (m *Git) BaseContainer() *dagger.Container {
	return dag.Container().
		From("alpine:latest").
		WithExec(helper.ForgeCommand("apk install --update git")).
		WithExec(helper.ForgeCommand("git config --global --add --bool push.autoSetupRemote true"))
}

// SetConfig permit to set git config
func (m *Git) SetConfig(
	// The git username
	username string,

	// The git email
	email string,
) *Git {
	m.baseContainer = m.baseContainer.
		WithExec(helper.ForgeCommandf("git config --global user.name %s", username)).
		WithExec(helper.ForgeCommandf("git config --global user.email %s", email))
	return m
}

// CommitAndPush permit to commit and push
func (m *Git) CommitAndPush(
	ctx context.Context,

	// the source directory
	source *dagger.Directory,

	// The commit message
	message string,
) (string, error) {

	return m.baseContainer.
		WithExec(helper.ForgeCommand("git add -A ")).
		WithExec(helper.ForgeScript(
			`
if [ -n "\$(git status --untracked-files=no --porcelain)" ]; then
	git commit -m "%s"
	git push
fi
		`, message)).
		Stdout(ctx)
}
