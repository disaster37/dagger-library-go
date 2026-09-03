// A generated module for GitModule functions
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
	"net/url"

	"dagger/git-module/internal/dagger"
	"fmt"

	"emperror.dev/errors"
	"github.com/disaster37/dagger-library-go/lib/v2/ci"
	"github.com/disaster37/dagger-library-go/lib/v2/helper"
)

type CI ci.CI

type GitModule struct {

	// +private
	Src *dagger.Directory

	Ci CI

	Container *dagger.Container
}

func New(
	// The source
	src *dagger.Directory,

	// base container
	// +optional
	baseContainer *dagger.Container,

	// The CI type
	// It permit to set the right commiter or right message to avoid loop on CI
	// +optional
	ci CI,
) *GitModule {
	git := &GitModule{
		Src: src,
		Ci:  ci,
	}
	if baseContainer != nil {
		git.Container = baseContainer
	} else {
		git.Container = dag.Container().
			From("alpine:latest").
			WithExec(helper.ForgeCommand("apk add --update git"))
	}

	git.Container = git.Container.
		WithDirectory("/source", src).
		WithWorkdir("/source").
		WithExec(helper.ForgeCommand("git config --global --add --bool push.autoSetupRemote true")).
		WithExec(helper.ForgeCommand("git config --global --add safe.directory /source"))

	return git
}

// GetCurrentBranchName Get the current branch name
func (m *GitModule) GetCurrentBranchName(
	ctx context.Context,
) (string, error) {
	return m.Container.
		WithExec(helper.ForgeCommand("git rev-parse --abbrev-ref HEAD")).
		Stdout(ctx)
}

// GetCurrentUrl get the origin URL
func (m *GitModule) GetCurrentUrl(
	ctx context.Context,
) (string, error) {
	return m.Container.
		WithExec(helper.ForgeCommand("git remote get-url origin")).
		Stdout(ctx)
}

// WithCustomContainer set a custom container
// It need to derive from the current container
func (m *GitModule) WithCustomContainer(c *dagger.Container) *GitModule {
	m.Container = c
	return m
}

// SetConfig permit to set git config
func (m *GitModule) SetConfig(
	ctx context.Context,

	// The git username
	// +optional
	username string,

	// The git email
	// +optional
	email string,

) (*GitModule, error) {

	if username == "" {
		switch m.Ci {
		case CI(ci.Github):
			username = string(ci.Github)
		case CI(ci.Jenkins):
			username = string(ci.Jenkins)
		default:
			username = string(ci.Dagger)
		}
	}

	if email == "" {
		switch m.Ci {
		case CI(ci.Github):
			email = fmt.Sprintf("%s@localhost", string(ci.Github))
		case CI(ci.Jenkins):
			email = fmt.Sprintf("%s@localhost", string(ci.Jenkins))
		default:
			email = fmt.Sprintf("%s@localhost", string(ci.Dagger))
		}
	}

	m.Container = m.Container.
		WithExec(helper.ForgeCommandf("git config --global user.name %s", username)).
		WithExec(helper.ForgeCommandf("git config --global user.email %s", email))

	return m, nil
}

// CommitAndPush permit to commit and push
func (m *GitModule) CommitAndPush(
	ctx context.Context,

	// The git repo URL where to commit
	// You need to provide it if you are currently on PullRequest
	// +optional
	gitRepoUrl string,

	// The branch name where to commit
	// You need to provide it if you are on PullRequest or in Tag
	// +optional
	branchName string,

	// The git token
	// +required
	token *dagger.Secret,

	// The commit message
	// +optional
	// +default="Commit from CI"
	message string,
) (string, error) {

	var err error

	// Configure git repo URL
	if gitRepoUrl == "" {
		gitRepoUrl, err = m.GetCurrentUrl(ctx)
		if err != nil {
			return "", errors.Wrap(err, "Error when try to get git URL from origin")
		}
	}
	u, err := url.Parse(gitRepoUrl)
	if err != nil {
		return "", errors.Wrap(err, "Error when par git URL")
	}

	// Configure credentials
	tokenPlain, err := token.Plaintext(ctx)
	if err != nil {
		return "", errors.Wrap(err, "Error when get token")
	}
	switch m.Ci {
	case CI(ci.Github):
		m.Container = m.Container.
			WithNewFile("/.git-credentials", fmt.Sprintf(`%s://%s@%s`, u.Scheme, tokenPlain, u.Host)).
			WithExec([]string{"git", "config", "--global", "credential.helper", "store --file /.git-credentials"})
	default:
		m.Container = m.Container.
			WithNewFile("/.git-credentials", fmt.Sprintf(`%s://git:%s@%s`, u.Scheme, tokenPlain, u.Host)).
			WithExec([]string{"git", "config", "--global", "credential.helper", "store --file /.git-credentials"})
	}

	// Configure git massage
	if message == "" {
		message = "Commit from CI"
	}
	if m.Ci == CI(ci.Github) {
		message = fmt.Sprintf("%s [skip ci]", message)
	}

	// Configure branch
	if branchName != "" {
		currentBranch, err := m.GetCurrentBranchName(ctx)
		if err != nil {
			return "", errors.Wrap(err, "Error when get current branch name")
		}

		if currentBranch != branchName {
			m.Container = m.Container.
				WithExec(helper.ForgeScript(`
git add -A
if [ -n "\$(git status --untracked-files=no --porcelain)" ]; then
	git commit -m "%s"
fi
git fetch origin %s:%s
git checkout %s
git merge %s
			`, message, branchName, branchName, branchName, currentBranch))
		}
	}

	return m.Container.
		WithExec(helper.ForgeCommand("git add -A")).
		WithExec(helper.ForgeScript(
			`
if [ -n "\$(git status --untracked-files=no --porcelain)" ]; then
	git commit -m "%s"
	git pull --rebase
	git push
fi
		`, message)).
		Stdout(ctx)
}
