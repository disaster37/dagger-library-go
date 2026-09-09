package main

import (
	"context"
	"fmt"
	"strings"

	"dagger/image/internal/dagger"

	"emperror.dev/errors"
	cimodule "github.com/disaster37/dagger-library-go/lib/v2/ci"
	"github.com/disaster37/dagger-library-go/lib/v2/pipeline"
)

type CI cimodule.CI

// Ci runs lint, build, and push (when enabled) as a single CI entrypoint.
func (m *Image) Ci(
	ctx context.Context,

	// Source directory
	// +required
	source *dagger.Directory,

	// Dockerfile path
	// +optional
	// +default="Dockerfile"
	dockerfile string,

	// Run in ci: push the image to the registry
	// +optional
	ci bool,

	// The registry server
	// +optional
	registry string,

	// The repository name
	// +optional
	repositoryName string,

	// The image version to publish
	// +optional
	version string,

	// The registry username
	// +optional
	registryUsername *dagger.Secret,

	// The registry password
	// +optional
	registryPassword *dagger.Secret,

	// Dry-run: skip the push and its requirement checks even when ci is set
	// +optional
	dryRun bool,
) (*dagger.Directory, error) {

	// Lint image
	stdout, err := m.Lint(ctx, source, dockerfile, "error")
	if err != nil {
		return nil, errors.Wrap(err, "Error when lint Dockerfile")
	}
	fmt.Println(stdout)

	// Build image, and force evaluation to catch build failures
	image := m.Build(source, dockerfile, nil)
	if _, err = image.GetContainer().Sync(ctx); err != nil {
		return nil, errors.Wrap(err, "Error when build image")
	}

	// Push image
	if ci && !dryRun {
		if registry == "" {
			return nil, errors.New("You must provide registry")
		}
		if repositoryName == "" {
			return nil, errors.New("You must provide repositoryName")
		}
		if version == "" {
			return nil, errors.New("You must provide the version")
		}
		if stdout, err = image.Push(
			ctx,
			repositoryName,
			version,
			registryUsername,
			registryPassword,
			registry,
		); err != nil {
			return nil, errors.Wrap(err, "Error when push image on registry")
		}
		fmt.Println(stdout)
	}

	return source, nil
}

// GenerateCi generates CI pipeline files for the given CI system.
func (m *Image) GenerateCi(
	ctx context.Context,

	// The CI runner: github, jenkins, or gitlab
	// +required
	ci CI,

	// Dockerfile path to lint and build
	// +optional
	// +default="Dockerfile"
	dockerfile string,

	// Branches that trigger the pipeline
	// +optional
	// +default=["main"]
	branches []string,

	// Dagger CLI version to use in CI (empty = engine default)
	// +optional
	daggerVersion string,

	// OCI registry URL where the image is pushed.
	// Required by the generated pipeline (it runs with --ci true).
	// +optional
	registry string,

	// Repository path inside the registry.
	// Required when registry is set.
	// +optional
	repository string,

	// Branch commits land here when running on a tag.
	// +optional
	// +default="main"
	defaultBranch string,

	// Configurable dagger module reference.
	// Defaults to this module's current version (auto-detected).
	// +optional
	moduleRef string,

	// GitHub: secret name for registry username (empty = github.actor).
	// +optional
	registryUsernameKey string,

	// GitHub: secret name for registry password (empty = GITHUB_TOKEN).
	// +optional
	registryPasswordKey string,

	// Jenkins: credential id for registry username/password.
	// +optional
	registryCredential string,

	// Jenkins: credential id for git token.
	// +optional
	gitTokenCredential string,

	// GitLab: CI/CD variable name for registry username.
	// +optional
	registryUsernameVar string,

	// GitLab: CI/CD variable name for registry password.
	// +optional
	registryPasswordVar string,

	// GitLab: CI/CD variable name for git token.
	// +optional
	gitTokenVar string,
) (*dagger.Directory, error) {
	var err error

	if len(branches) == 0 {
		branches = []string{"main"}
	}
	if daggerVersion == "" {
		daggerVersion, err = dag.Version(ctx)
		if err != nil {
			return nil, err
		}
	}
	if defaultBranch == "" {
		defaultBranch = "main"
	}
	if moduleRef == "" {
		moduleRef = fmt.Sprintf("github.com/disaster37/dagger-library-go/image@%s", strings.TrimSpace(ModuleVersion))
	}

	// Determine placeholder bindings based on CI
	registryUserBinding, registryPassBinding, gitTokenBinding, err := pipeline.ResolveCredentialBindings(cimodule.CI(ci), pipeline.CredentialConfig{
		RegistryUsernameKey: registryUsernameKey,
		RegistryPasswordKey: registryPasswordKey,
		RegistryCredential:  registryCredential,
		GitTokenCredential:  gitTokenCredential,
		RegistryUsernameVar: registryUsernameVar,
		RegistryPasswordVar: registryPasswordVar,
		GitTokenVar:         gitTokenVar,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Error when resolve credential bindings")
	}

	// Build PipelineSpec
	spec := pipeline.PipelineSpec{
		CI:            cimodule.CI(ci),
		ModuleRef:     moduleRef,
		DaggerVersion: daggerVersion,
		Branches:      branches,
		DefaultBranch: defaultBranch,
		Triggers: pipeline.Triggers{
			Push:        true,
			PullRequest: true,
			Tag:         true,
		},
		Job: pipeline.Job{
			Function: "ci",
			Args: []string{
				"--source", ".",
				"--dockerfile", dockerfile,
				"--ci=true",
				"--registry", registry,
				"--repository-name", repository,
				"--version", "{{version}}",
				"--registry-username", "{{registry-username}}",
				"--registry-password", "{{registry-password}}",
			},
			Placeholders: map[string]pipeline.Binding{
				pipeline.PhVersion:      {Kind: pipeline.BindingExpr, Ref: ""},
				pipeline.PhRegistryUser: registryUserBinding,
				pipeline.PhRegistryPass: registryPassBinding,
				pipeline.PhGitToken:     gitTokenBinding,
			},
		},
		Registry:   registry,
		Repository: repository,
	}

	files, err := pipeline.Render(spec)
	if err != nil {
		return nil, errors.Wrap(err, "Error when render CI pipeline")
	}

	dir := dag.Directory()
	for path, content := range files {
		dir = dir.WithNewFile(path, content)
	}

	return dir, nil
}
