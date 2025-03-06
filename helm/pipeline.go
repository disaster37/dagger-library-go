package main

import (
	"context"
	"dagger/helm/internal/dagger"

	"emperror.dev/errors"
	"github.com/disaster37/dagger-library-go/lib/ci"
)

type CI ci.CI

func (m *Helm) Ci(
	ctx context.Context,

	// The registry
	registry string,

	// The repository inside the registry
	repository string,

	// The helm paths
	// +default=["."]
	helmPaths []string,

	// If you are on CI, provide the CI type to set the right stuff when commit to ovoid loop for ever
	// +optional
	ci CI,

	// The image version to publish
	// +optional
	version string,

	// The registry username
	registryUsername *dagger.Secret,

	// The registry password
	registryPassword *dagger.Secret,

	// The git token
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

	if len(helmPaths) == 0 {
		helmPaths = []string{"."}
	}

	rootDir := m.Src

	// Add repository
	m = m.WithRepository(
		ctx,
		repository,
		registry,
		true,
		registryUsername,
		registryPassword,
	)

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
