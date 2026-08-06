# Generic CI Pipeline Generator for dagger-library-go

## Goal

Replace the helm-only, quicktemplate-based CI YAML generation with a **generic, pure-Go library** (`lib/pipeline`) that programmatically builds CI pipelines for **GitHub Actions, Jenkins, and GitLab CI** (with a renderer interface that makes CircleCI a trivial follow-up). The library is reusable by every module in this repo (helm, image, golang, k3s, kwok, operator-sdk) and removes the hardcoded module path, moves version-strategy into testable Go, and adds dry-run support.

## Non-goals

- CircleCI renderer (interface supports it; out of scope for this pass).
- Generating pipelines for modules that have no publish/release story (k3s, kwok are runtime-only cluster helpers — they get no `GenerateCi`).
- Customizing the generated pipeline beyond the exposed spec (no user-supplied step injection).
- Changing the runtime `Ci`/`Release` execution logic of modules — only their `GenerateCi` surface and the dry-run flag.
- Replacing the `git-module` commit/push behavior.

## Architecture decision: pure library, not a Dagger module

`lib/pipeline` is a **pure Go package with zero Dagger SDK imports**. Rationale:

1. **Testable without a Dagger engine** — renderers and version logic are unit-testable with `go test` and golden YAML files. A Dagger-module approach would require the engine for every test.
2. **No version coupling** — every consumer module already `replace`s `lib => ../lib/`. Adding a new shared Dagger module would force all consumers to add a Dagger dependency and pin a version, recreating the hardcoded-`@v2` problem at a different layer.
3. **Single source of truth** — the YAML/CI-expression logic lives in one package imported by all modules, instead of being duplicated as templates per module.

Each module exposes a thin Dagger function `GenerateCi` that builds a `pipeline.PipelineSpec` from its Dagger args, calls `pipeline.Render`, and writes the returned files into a `*dagger.Directory`. The Dagger SDK types stay in the module layer; `lib/pipeline` only deals with strings, maps, and `gopkg.in/yaml.v3`.

## Directory layout

### New files

```
lib/pipeline/
  ci.go              # CI type re-export + supported set
  spec.go            # PipelineSpec, Job, Binding, Triggers, VersionStrategy, constants
  validate.go        # Validate(spec) error
  version.go         # VersionStrategy, VersionContext, ComputeVersion (testable)
  command.go         # assembleShellCommand(spec) (placeholder -> $ENV substitution)
  render.go          # Renderer interface, Renderers registry, Render(spec)
  renderer_github.go # GitHubRenderer (YAML via yaml.v3)
  renderer_jenkins.go# JenkinsRenderer (Groovy via structured emitter)
  renderer_gitlab.go # GitLabRenderer (YAML via yaml.v3)
  dagger_md.go       # RenderDaggerMd(spec) string (generic local-usage doc)
  groovy.go          # small structured Groovy string builder w/ escaping
  spec_test.go
  validate_test.go
  version_test.go
  command_test.go
  render_test.go     # golden-file tests per renderer
  testdata/
    github_default.yaml
    github_prerelease.yaml
    jenkins_default.groovy
    gitlab_default.yaml
    dagger_md_default.md
```

### Modified files

```
lib/go.mod                       # add gopkg.in/yaml.v3 as direct require
lib/ci/ci.go                     # add Gitlab CI = "gitlab" constant
helm/pipeline.go                 # rewrite GenerateCi to use lib/pipeline; add dryRun to Ci; panics -> errors
helm/main.go                     # remove //go:generate qtc directive
helm/go.mod                      # drop github.com/valyala/quicktemplate; keep yaml.v3 (now from lib)
helm/README.md                   # update GenerateCi docs (module-ref param, gitlab, dry-run)
golang/main.go (or new golang/pipeline.go)  # add GenerateCi
image/main.go    (or new image/pipeline.go)  # add GenerateCi
operator-sdk/main.go (or new operator-sdk/pipeline.go) # add GenerateCi
golang/README.md, image/README.md, operator-sdk/README.md # document GenerateCi
```

### Deleted files

```
helm/templates/params.go
helm/templates/github.qtpl
helm/templates/github.qtpl.go
helm/templates/jenkins.qtpl
helm/templates/jenkins.qtpl.go
helm/templates/dagger.md.qtpl
helm/templates/dagger.md.qtpl.go
helm/templates/   (whole directory)
```

## Data structures (exact Go)

```go
// lib/pipeline/ci.go
package pipeline

import "github.com/disaster37/dagger-library-go/lib/ci"

// Supported returns the CI systems this library can render.
func Supported() []ci.CI { return []ci.CI{ci.Github, ci.Jenkins, ci.GitLab} }
```

```go
// lib/pipeline/spec.go
package pipeline

import "github.com/disaster37/dagger-library-go/lib/ci"

// PipelineSpec is a CI-agnostic description of a release/CI pipeline.
// All fields are plain Go types — no Dagger SDK types, so the package is
// unit-testable without a Dagger engine.
type PipelineSpec struct {
    CI             ci.CI
    ModuleRef      string         // e.g. "github.com/disaster37/dagger-library-go/helm@v2" (required)
    DaggerVersion  string         // empty = engine default
    Branches       []string       // default ["main"]
    DefaultBranch  string         // default "main"; branch commits land here on tag
    Triggers       Triggers
    Job            Job
    Registry       string         // empty = no push; renderer omits push flags
    Repository     string         // required when Registry != ""
    VersionStrategy VersionStrategy
    TimeoutMinutes  int           // 0 = renderer default (GitHub: none, Jenkins: 10, GitLab: 30)
    ExtraFiles      map[string]string // additional filename->content emitted alongside
}

type Triggers struct {
    Push        bool // default true
    PullRequest bool // default true
    Tag         bool // default true
    Release     bool // default false (GitHub only)
}

// Job describes the `dagger call` invocation. Args values may contain
// placeholders listed in Placeholders; renderers substitute each with a
// shell env-var reference ($NAME) and emit the matching env/secret block.
type Job struct {
    Function     string              // e.g. "ci", "release" (required)
    Args         []string            // ordered flags, e.g. ["--src",".","--ci","github","--version","{{version}}"]
    Placeholders map[string]Binding  // canonical + custom placeholder bindings
}

// BindingKind selects how a placeholder value is sourced at CI runtime.
type BindingKind int

const (
    BindingSecret BindingKind = iota // Ref = CI secret/credential name
    BindingExpr                       // Ref = CI-native expression (renderer must support)
    BindingLiteral                    // Ref = plain string, emitted verbatim
)

type Binding struct {
    Kind BindingKind
    Ref  string
}

// Canonical placeholder names. Modules use these in Job.Args and the
// renderer wires them up per-CI automatically.
const (
    PhVersion       = "version"
    PhBranch        = "branch"
    PhGitRepoURL    = "git-repo-url"
    PhGitToken      = "git-token"
    PhRegistryUser  = "registry-username"
    PhRegistryPass  = "registry-password"
)
```

```go
// lib/pipeline/version.go
package pipeline

// VersionStrategy describes how the release version is computed from CI
// context at runtime. Patterns use {build}, {pr}, {tag}, {branch} tokens.
type VersionStrategy struct {
    BranchPattern    string // default "0.0.0-rc.{build}"
    PRPattern        string // default "0.0.0-pr.{pr}.{build}"
    TagPattern       string // default "{tag}"
    PrereleaseSuffix string // optional; appended to a bumped base version (helm prerelease behavior)
}

// VersionContext is the CI runtime context. Used by ComputeVersion for
// testing/dry-run; renderers translate the same strategy into native CI
// expressions.
type VersionContext struct {
    Event    string // "push" | "pull_request" | "tag" | "release"
    Build    int
    PRNumber int
    Tag      string
    Branch   string
}

// ComputeVersion resolves a strategy against a concrete context. Pure,
// deterministic, unit-testable. Returns error on unknown event.
func ComputeVersion(s VersionStrategy, ctx VersionContext) (string, error)
```

```go
// lib/pipeline/render.go
package pipeline

import "github.com/disaster37/dagger-library-go/lib/ci"

// Renderer turns a PipelineSpec into a map of output-path -> content.
type Renderer interface {
    Render(spec PipelineSpec) (map[string]string, error)
    Supports(c ci.CI) bool
}

var Renderers = map[ci.CI]Renderer{
    ci.Github:  &GitHubRenderer{},
    ci.Jenkins: &JenkinsRenderer{},
    ci.GitLab:  &GitLabRenderer{},
}

// Render validates the spec and dispatches to the registered renderer.
func Render(spec PipelineSpec) (map[string]string, error)

// RenderDaggerMd produces a generic DAGGER.md local-usage doc using
// env:VAR placeholders for secrets. Added to every render output.
func RenderDaggerMd(spec PipelineSpec) string
```

```go
// lib/pipeline/validate.go
package pipeline

// Validate enforces required fields and applies defaults. Returns error
// (never panics). Rules:
//   - CI required and must be in Supported()
//   - ModuleRef required, must contain "@"
//   - Branches default ["main"] if empty
//   - DefaultBranch default "main" if empty
//   - Triggers: if all false, set Push+PullRequest+Tag true
//   - Job.Function required
//   - If Registry != "": Repository required; PhRegistryUser, PhRegistryPass,
//     PhGitToken bindings must be present and of kind BindingSecret
//   - VersionStrategy defaults applied for empty patterns
func Validate(spec *PipelineSpec) error // takes pointer so it can apply defaults
```

```go
// lib/pipeline/command.go
package pipeline

// assembleShellCommand builds the `dagger call -m <ref> <function> <args...>`
// shell string with placeholders replaced by $ENV_VAR references.
// It returns (commandString, envDecls) where envDecls is the ordered
// list of (envVar, binding) the renderer must emit in its env block.
func assembleShellCommand(spec PipelineSpec) (cmd string, envDecls []envDecl, err error)

type envDecl struct {
    EnvVar  string
    Binding Binding
}
```

## Function signatures (exact)

### Library (lib/pipeline)

```go
func Supported() []ci.CI
func Render(spec PipelineSpec) (map[string]string, error)
func RenderDaggerMd(spec PipelineSpec) string
func Validate(spec *PipelineSpec) error
func ComputeVersion(s VersionStrategy, ctx VersionContext) (string, error)
func assembleShellCommand(spec PipelineSpec) (cmd string, envDecls []envDecl, err error) // unexported
```

### Helm Dagger function (migrated)

```go
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
    // +optional
    // +default="github.com/disaster37/dagger-library-go/helm@v2"
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
) (*dagger.Directory, error)
```

`Ci` gains:

```go
    // Dry-run: skip push and git commit/push even when --ci is set.
    // +optional
    dryRun bool,
```

### Generic module adoption (golang, image, operator-sdk)

Each adds a `GenerateCi` with the same parameter shape as helm's (minus helm-specific `helmPaths`, plus module-specific path args). Example for golang:

```go
func (m *Golang) GenerateCi(
    ctx context.Context,
    ci CI,
    branches []string,
    daggerVersion string,
    defaultBranch string,
    moduleRef string,        // +default="github.com/disaster37/dagger-library-go/golang@v2"
    registryUsernameKey string,
    registryPasswordKey string,
    registryCredential string,
    gitTokenCredential string,
    registryUsernameVar string,
    registryPasswordVar string,
    gitTokenVar string,
) (*dagger.Directory, error)
```

The module's `GenerateCi` builds a `PipelineSpec` whose `Job.Function` is the module's release entrypoint (helm: `ci`; golang: a new `ci` wrapper or `lint`+`test`+`build`; operator-sdk: `release`) and `Job.Args` carry the module-specific flags with canonical placeholders.

## Migration of helm/GenerateCi and templates

1. **Add `lib/pipeline`** with all types, renderers, validation, version, command assembly, and golden tests. Add `gopkg.in/yaml.v3` to `lib/go.mod` require. Add `Gitlab` to `lib/ci/ci.go`.
2. **Rewrite `helm/pipeline.go` `GenerateCi`**: construct a `PipelineSpec` from the Dagger args (mapping `registryUsernameKey`→`Binding{BindingSecret, "GITHUB_TOKEN"}` etc. per CI), call `pipeline.Render`, write each returned file into `dag.Directory()`. The `DAGGER.md` file comes from `pipeline.RenderDaggerMd` (included in the render output map).
3. **Add `dryRun bool` to `Helm.Ci`**: when true, skip the `m.Push` block and the `git-module.CommitAndPush` block, but still run lint/schema/docs. Convert the existing `panic(...)` calls in `Ci` to `return nil, errors.New(...)`.
4. **Delete `helm/templates/`** entirely and remove the `//go:generate qtc` directive in `helm/main.go`. Remove `github.com/valyala/quicktemplate` from `helm/go.mod`.
5. **Adopt in other modules**: add `GenerateCi` to `golang`, `image`, `operator-sdk` (each in a new `pipeline.go` in the module dir to keep `main.go` focused). k3s/kwok get nothing (runtime-only).
6. **Update READMEs** to document `--module-ref`, `--dry-run`, and the new `gitlab` CI option.

## Edge cases

- **No branches supplied** → `Validate` defaults to `["main"]`.
- **Multiple helm paths** → `Job.Args` carries repeated `--helm-paths` flags; renderer just substitutes placeholders, so multi-path works unchanged.
- **Tag vs PR vs branch versioning** → `VersionStrategy` patterns; each renderer emits the native CI expression selecting among the three. `ComputeVersion` is unit-tested for all three events.
- **Prerelease suffix (helm)** → `VersionStrategy.PrereleaseSuffix`; the helm `Ci` runtime already bumps patch+prerelease in Go — that stays in `Ci`, not in the generator. The generator only emits the *base* version expression; the helm `Ci` function applies the prerelease bump at runtime. Documented in the spec.
- **Private registries** → `Registry` empty disables push; renderer omits `--registry`/`--repository`/credential flags and the push step. When set, credentials are required by `Validate`.
- **OCI vs classic helm repos** → not a generator concern; `Ci`/`Push` handle OCI. Generator only passes `--registry`/`--repository`.
- **Custom containers** → out of scope; module-level, not in CI YAML.
- **Secrets vs env vars** → `BindingKind` distinguishes `BindingSecret` (CI secret/credential) from `BindingExpr` (CI-native expression) from `BindingLiteral`.
- **No-op commits** → `git-module.CommitAndPush` already skips empty diffs; `dryRun` skips the call entirely.
- **Empty values.yaml** → helm `Ci` already skips schema/readme; generator unaffected.
- **Unsupported CI** → `Render` returns `fmt.Errorf("unsupported CI: %s", spec.CI)`.
- **Missing module ref** → `Validate` returns error; Dagger function default covers the common case.
- **Tag with no default branch** → `Validate` defaults `DefaultBranch` to `main`.
- **GitHub release trigger** → only emitted when `Triggers.Release` true (default false); other CI systems ignore it.

## Error handling and validation

- **Library never panics.** All public functions return `error`.
- `Validate` applies defaults in place (pointer receiver) and returns errors for: missing CI, unsupported CI, missing/invalid ModuleRef, missing Job.Function, Registry set without Repository, Registry set without the three required secret bindings.
- Dagger `GenerateCi` functions return errors from `pipeline.Render`; they do not panic.
- Helm `Ci` existing `panic` calls are converted to `return nil, errors.New(...)` (uses `emperror.dev/errors` consistent with the rest of the file).
- `ComputeVersion` returns error on unknown `Event`.

## Testing strategy

### Unit tests (lib/pipeline, no Dagger engine needed)

- `validate_test.go`: defaults applied; each validation rule rejects bad input.
- `version_test.go`: `ComputeVersion` for push/PR/tag/release events; custom patterns; prerelease suffix; unknown event error.
- `command_test.go`: `assembleShellCommand` substitutes all canonical placeholders; unknown placeholder → error; literal binding emitted verbatim; ordering preserved.
- `render_test.go`: golden-file comparison for each renderer:
  - `github_default.yaml`, `github_prerelease.yaml` (with registry + custom secrets)
  - `jenkins_default.groovy`, `jenkins_multi_branch.groovy`
  - `gitlab_default.yaml`
  - `dagger_md_default.md`
  - Each test builds a `PipelineSpec`, calls `Render`, and compares the output map to golden files via `cmp.Diff`. Golden files are committed and updated with `-update` flag.
- `renderer_github_test.go` / `renderer_jenkins_test.go` / `renderer_gitlab_test.go`: per-renderer edge cases (no registry, custom secret names, release trigger, timeout override, multiple branches).

### Build/vet/test commands

Run from repo root and each module:

```bash
cd lib && go build ./... && go vet ./... && go test ./...
cd helm && go build ./... && go vet ./... && go test ./...
cd golang && go build ./... && go vet ./... && go test ./...
cd image && go build ./... && go vet ./... && go test ./...
cd operator-sdk && go build ./... && go vet ./... && go test ./...
```

Golden update: `cd lib && go test ./pipeline/ -update`.

### Integration check (manual, not automated in this pass)

After migration, regenerate helm's pipeline from a sample chart and diff against the pre-migration output to confirm behavioral parity for GitHub + Jenkins (GitLab is net-new).

## Migration / rollout order

1. **lib/pipeline skeleton + tests pass** (no module changes yet). Acceptance: `go test ./...` in `lib` green; golden files committed.
2. **helm migration**: rewrite `GenerateCi`, add `dryRun` to `Ci`, convert panics, delete `helm/templates/`, drop quicktemplate dep. Acceptance: `go build ./... && go vet ./... && go test ./...` in `helm` green; `dagger call -m helm --src . generate-ci --ci github export --path .` produces a workflow byte-identical (modulo formatting) to the old template output; `--ci gitlab` produces a valid `.gitlab-ci.yml`; `--module-ref` overrides the ref in output; `ci --dry-run` skips push.
3. **golang/image/operator-sdk adoption**: add `GenerateCi` + `pipeline.go` per module. Acceptance: each module builds, vets, tests; `dagger call -m <module> generate-ci --ci github export --path .` produces a valid workflow invoking that module's release function.
4. **README updates** for all touched modules.

## Risks and open questions

1. **Jenkins Groovy emission** — Jenkins uses declarative Groovy, not YAML. A small structured Groovy emitter (`groovy.go`) with proper string escaping is needed. Risk: Groovy edge cases (quoting, GString interpolation collisions with `${...}`). Mitigation: the emitter escapes `$` and quotes; golden tests lock the output.
2. **Version expression fidelity** — moving from hand-written bash one-liners to pattern-driven CI expressions risks subtle behavior changes (e.g. GitHub `github.ref_type == 'tag'` precedence). Mitigation: golden files captured from the current templates as the parity baseline for GitHub/Jenkins; GitLab is new and reviewed manually.
3. **lib dependency surface** — adding `yaml.v3` to `lib` pulls it into every consumer's build. Already transitively present in `lib/go.sum`, so low risk, but every module's `go.sum` will change.
4. **Per-module release entrypoints differ** — helm uses `ci`, operator-sdk uses `release`, golang/image have no single release function today. Open question: do golang/image need a new `ci` wrapper function, or should `GenerateCi` emit a multi-step pipeline calling `lint`, `test`, `build` sequentially? **Recommendation: emit a single `ci`-style entrypoint per module; for golang/image add a thin `Ci` Dagger function that chains lint+test+build, mirroring helm's `Ci`.** This keeps the generator uniform. Flagged for confirmation.
5. **Dagger version pinning in generated pipelines** — current GitHub template pins `dagger/dagger-for-github@v7`. The README notes v8+ uses `dagger run`. Open question: bump to v8 in the new renderer, or preserve v7 for parity? **Recommendation: preserve v7 in this pass (parity first), open a follow-up to migrate to `dagger run`.**
6. **GitLab secret variable naming** — GitLab uses CI/CD variables (not "secrets"). The `*Var` params map to CI/CD variable names. Confirm naming convention with the user; default to `REGISTRY_USERNAME` / `REGISTRY_PASSWORD` / `GIT_TOKEN`.
