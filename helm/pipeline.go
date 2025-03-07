package main

import (
	"context"
	"dagger/helm/internal/dagger"
	"dagger/helm/templates"

	"emperror.dev/errors"
	cimodule "github.com/disaster37/dagger-library-go/lib/ci"
)

type CI cimodule.CI

func (m *Helm) Ci(
	ctx context.Context,

	// The registry
	// You need to provide it if run from CI
	// +optional
	registry string,

	// The repository inside the registry
	// You need to provide it if run from CI
	// +optional
	repository string,

	// The helm paths
	// +optional
	// +default=["."]
	helmPaths []string,

	// If you are on CI, provide the CI type to set the right stuff when commit to ovoid loop for ever
	// +optional
	ci CI,

	// The image version to publish
	// You need to provide it if run from CI
	// +optional
	version string,

	// The registry username
	// You need to provide it if run from CI
	// +optional
	registryUsername *dagger.Secret,

	// The registry password
	// You need to provide it if run from CI
	// +optional
	registryPassword *dagger.Secret,

	// The git token
	// You need to provide it if run from CI
	// +optional
	gitToken *dagger.Secret,

	// The git repo URL
	// You need to provide it when you are on PullRequest
	// +optional
	gitRepoUrl string,

	// The git branch where you should to push
	// You need to provide it when you are on PullRequest or on Tag
	// +optional
	gitBranch string,
) (dir *dagger.Directory, err error) {

	if ci != "" {
		if registry == "" {
			panic("You need to set registry")
		}
		if repository == "" {
			panic("You need to set repository")
		}
		if version == "" {
			panic("You need to set version")
		}
		if registryUsername == nil {
			panic("You need to set registry-username")
		}
		if registryPassword == nil {
			panic("you need to set registry-password")
		}
		if gitToken == nil {
			panic("you need to set git-token")
		}
		if gitRepoUrl == "" {
			panic("you need to set git-repo-url")
		}
	}

	if len(helmPaths) == 0 {
		helmPaths = []string{"."}
	}

	rootDir := m.Src

	// Add repository
	if registry != "" {
		m = m.WithRepository(
			ctx,
			repository,
			registry,
			true,
			registryUsername,
			registryPassword,
		)
	}

	for _, helmPath := range helmPaths {

		m = m.WithSource(rootDir.Directory(helmPath))

		// Generate helm schema
		schemaFile, err := m.GenerateSchema("", "")
		if err != nil {
			return nil, errors.Wrap(err, "Error when generate schema")
		}
		filename, err := schemaFile.Name(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "Error when get file name")
		}
		m = m.WithSource(m.Src.WithFile(filename, schemaFile))

		// Generate readme
		readmeFile, err := m.GenerateDocumentation("", "")
		if err != nil {
			return nil, errors.Wrap(err, "Error when generate documentation")
		}
		filename, err = readmeFile.Name(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "Error when get file name")
		}
		m = m.WithSource(m.Src.WithFile(filename, readmeFile))

		//Lint helm chart
		_, err = m.Lint(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "Error when lint chart")
		}

		// Push helm chart
		if ci != "" {
			chartFile, err := m.Push(
				ctx,
				registry,
				repository,
				version,
			)
			if err != nil {
				return nil, errors.Wrap(err, "Error when push helm chart")
			}
			filename, err = chartFile.Name(ctx)
			if err != nil {
				return nil, errors.Wrap(err, "Error when get file name")
			}
			m = m.WithSource(m.Src.WithFile(filename, chartFile))
		}

		rootDir = rootDir.WithDirectory(helmPath, m.Src)
	}

	// Commit and push
	if ci != "" {

		_, err = dag.GitModule(
			rootDir,
			dagger.GitModuleOpts{
				Ci: string(ci),
			},
		).
			SetConfig().
			CommitAndPush(
				ctx,
				gitToken,
				dagger.GitModuleCommitAndPushOpts{
					GitRepoURL: gitRepoUrl,
					BranchName: gitBranch,
				},
			)
		if err != nil {
			return nil, errors.Wrap(err, "Error when commit/push")
		}
	}

	return rootDir, nil

}

// GenerateCi permit to generate CI file
func (m *Helm) GenerateCi(
	ctx context.Context,

	// The CI runner
	ci CI,

	// The branch from CI
	// +default=["main"]
	branches []string,

	// The dagger version to use
	// Only used with Github
	// Default use the current dagger version
	// +optional
	daggerVersion string,

	// The registry where to push helm chart
	// Push to ghcr.io by default when you are on github CI
	// +optional
	registry string,

	// The repository
	// Push to current repository by default when you are on github CI
	// +optional
	repository string,

	// The default branch name
	// It's needed to commit change from a tag
	// +optional
	// +default="main"
	defaultBranch string,

	// The registry credential name
	// Only used when Jenkins pipeline
	// +optional
	registryCredential string,

	// The credential name for registry username
	// Only used when Github pipeline
	// Default it use the current user
	// +optional
	registryUsernameKey string,

	// The credential name for registry password
	// Only used when Github pipeline
	// Default it use the current git token
	// +optional
	registryPasswordKey string,

	// The credential name for git token
	// Only used for Jenkins pipeline
	// +optional
	gitTokenCredential string,

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

	dir := dag.Directory()

	switch ci {
	case CI(cimodule.Github):
		fCi := templates.GenerateGithub(
			branches,
			templates.Opts{
				DaggerVersion:              daggerVersion,
				Registry:                   registry,
				Repository:                 repository,
				DefaultBranchName:          defaultBranch,
				RegistryUsernameSecretName: registryUsernameKey,
				RegistryPasswordSecretName: registryPasswordKey,
			},
		)

		dir = dir.WithNewFile(".github/workflows/dagger.yaml", fCi)
	case CI(cimodule.Jenkins):
		fCi := templates.GenerateJenkins(
			branches,
			templates.Opts{
				DaggerVersion:      daggerVersion,
				Registry:           registry,
				Repository:         repository,
				DefaultBranchName:  defaultBranch,
				RegistryCredential: registryCredential,
				GitTokenCredential: gitTokenCredential,
			},
		)

		dir = dir.WithNewFile("Jenkinsfile", fCi)
	default:
		return nil, errors.New("CI not supported")
	}

	return dir, nil
}
