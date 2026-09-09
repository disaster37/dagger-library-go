package main

import (
	"context"
	"fmt"
	"strings"

	"dagger/helm/internal/dagger"

	"emperror.dev/errors"
	"github.com/Masterminds/semver/v3"
	cimodule "github.com/disaster37/dagger-library-go/lib/v2/ci"
	"github.com/disaster37/dagger-library-go/lib/v2/pipeline"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chart"
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

	// Dry-run: skip push and git commit/push even when --ci is set.
	// +optional
	dryRun bool,
) (dir *dagger.Directory, err error) {

	var filename string

	isCI := ci != ""
	isGitPush := isCI && !dryRun

	if isCI {
		if registry == "" {
			return nil, errors.New("You need to set registry")
		}
		if repository == "" {
			return nil, errors.New("You need to set repository")
		}
		if version == "" {
			return nil, errors.New("You need to set version")
		}
		if registryUsername == nil {
			return nil, errors.New("You need to set registry-username")
		}
		if registryPassword == nil {
			return nil, errors.New("you need to set registry-password")
		}
		if gitToken == nil {
			return nil, errors.New("you need to set git-token")
		}
		if gitRepoUrl == "" {
			return nil, errors.New("you need to set git-repo-url")
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

	var ociUrls []string

	for _, helmPath := range helmPaths {

		// Init state
		currentHelmModule := m.WithWorkDir(sourceDirectory).WithSource(rootDir).WithWorkDir(helmPath)

		// Read Chart.yaml to get chart metadata
		fChart, err := currentHelmModule.Src.File("Chart.yaml").Contents(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "File 'Chart.yaml' not found")
		}
		chartMeta := &chart.Metadata{}
		if err := yaml.Unmarshal([]byte(fChart), chartMeta); err != nil {
			return nil, errors.Wrap(err, "Error when unmarshall 'Chart.yaml'")
		}

		// Forge target version
		localVersion := version
		v, err := semver.StrictNewVersion(version)
		if err != nil {
			return nil, errors.Wrap(err, "The version is not semver")
		}
		if v.Prerelease() != "" {
			vTarget, err := semver.StrictNewVersion(chartMeta.Version)
			if err != nil {
				return nil, errors.Wrap(err, "Error when convert to semver the current helm version")
			}
			*vTarget = vTarget.IncPatch()
			*vTarget, err = vTarget.SetPrerelease(v.Prerelease())
			if err != nil {
				return nil, errors.Wrap(err, "Error when forge next release version")
			}
			localVersion = vTarget.String()
		}

		// Track OCI URL for summary
		if isCI {
			ociUrls = append(ociUrls, fmt.Sprintf("oci://%s/%s/%s:%s", registry, repository, chartMeta.Name, localVersion))
		}

		// Skip Schema and readme if values.yaml not exist
		if _, err := currentHelmModule.Src.File("values.yaml").Sync(ctx); err == nil {
			// Generate helm schema
			schemaFile, err := currentHelmModule.GenerateSchema("", "")
			if err != nil {
				return nil, errors.Wrap(err, "Error when generate schema")
			}
			filename, err = schemaFile.Name(ctx)
			if err != nil {
				return nil, errors.Wrap(err, "Error when get file name")
			}
			currentHelmModule = currentHelmModule.WithSource(currentHelmModule.Src.WithFile(filename, schemaFile))

			// Generate readme
			readmeFile, err := currentHelmModule.GenerateDocumentation("", "")
			if err != nil {
				return nil, errors.Wrap(err, "Error when generate documentation")
			}
			filename, err = readmeFile.Name(ctx)
			if err != nil {
				return nil, errors.Wrap(err, "Error when get file name")
			}
			currentHelmModule = currentHelmModule.WithSource(currentHelmModule.Src.WithFile(filename, readmeFile))

		}

		//Lint helm chart
		_, err = currentHelmModule.Lint(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "Error when lint chart")
		}

		// Push helm chart
		if isCI {
			chartFile, err := currentHelmModule.Push(
				ctx,
				registry,
				repository,
				localVersion,
			)
			if err != nil {
				return nil, errors.Wrap(err, "Error when push helm chart")
			}
			filename, err = chartFile.Name(ctx)
			if err != nil {
				return nil, errors.Wrap(err, "Error when get file name")
			}

			// Add chartFile
			currentHelmModule = currentHelmModule.WithSource(currentHelmModule.Src.WithFile(filename, chartFile))
		}

		rootDir = rootDir.WithDirectory(helmPath, currentHelmModule.Src)
	}

	// Commit and push
	if isGitPush {

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

	// Print OCI URLs summary
	if len(ociUrls) > 0 {
		for _, url := range ociUrls {
			fmt.Println(url)
		}
	}

	return rootDir, nil

}

// GenerateCi generates CI pipeline files for the given CI system.
func (m *Helm) GenerateCi(
	ctx context.Context,

	// The CI runner: github, jenkins, or gitlab
	// +required
	ci CI,

	// Branches that trigger the pipeline
	// +optional
	// +default=["main"]
	branches []string,

	// Helm chart directory paths
	// +optional
	// +default=["."]
	helmPaths []string,

	// Dagger CLI version to use in CI (empty = engine default)
	// +optional
	daggerVersion string,

	// OCI registry URL. Empty = renderer default (ghcr.io on GitHub).
	// +optional
	registry string,

	// Repository path inside the registry.
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
		moduleRef = fmt.Sprintf("github.com/disaster37/dagger-library-go/helm@%s", strings.TrimSpace(ModuleVersion))
	}
	if len(helmPaths) == 0 {
		helmPaths = []string{"."}
	}

	// Build helm path flags
	var helmPathsArgs []string
	for _, p := range helmPaths {
		helmPathsArgs = append(helmPathsArgs, "--helm-paths", p)
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
		SrcDir:        ".",
		Triggers: pipeline.Triggers{
			Push:        true,
			PullRequest: true,
			Tag:         true,
		},
		Job: pipeline.Job{
			Function: "ci",
			Args: append([]string{
				"--registry", registry,
				"--repository", repository,
				"--ci", string(ci),
				"--version", "{{version}}",
				"--registry-username", "{{registry-username}}",
				"--registry-password", "{{registry-password}}",
				"--git-token", "{{git-token}}",
				"--git-repo-url", "{{git-repo-url}}",
				"--git-branch", "{{branch}}",
			}, helmPathsArgs...),
			Placeholders: map[string]pipeline.Binding{
				pipeline.PhVersion:      {Kind: pipeline.BindingExpr, Ref: ""},
				pipeline.PhRegistryUser:  registryUserBinding,
				pipeline.PhRegistryPass:  registryPassBinding,
				pipeline.PhGitToken:      gitTokenBinding,
				pipeline.PhGitRepoURL:   {Kind: pipeline.BindingExpr, Ref: ""},
				pipeline.PhBranch:       {Kind: pipeline.BindingExpr, Ref: ""},
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
