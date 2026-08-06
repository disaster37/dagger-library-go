# Helm README Generator Migration Plan

Migrate the `helm/` Dagger module's README/schema generator from the archived
Bitnami `@bitnami/readme-generator-for-helm` (npm, JS) to the maintained Go fork
`github.com/disaster37/readme-generator-for-helm`, and add a new Dagger function
that migrates `values.yaml` metadata tags to the new array syntax.

---

## 1. Goal

Replace the default `GeneratorContainer` (built from `node:21-alpine` + `npm
install -g @bitnami/readme-generator-for-helm`) with the prebuilt Docker image of
the Go fork, update the two existing generator functions to the fork's new
subcommand-based CLI, and add a new `MigrateValuesTags` function that wraps the
fork's `migrate` subcommand.

## 2. Scope

### Files to modify
- `helm/main.go` — replace default `GeneratorContainer` base image.
- `helm/generate_doc.go` — update `readme-generator` invocation to new CLI.
- `helm/generate_schema.go` — update `readme-generator` invocation to new CLI
  (uses the same `GeneratorContainer`; the old invocation would break with the
  new binary, so it MUST be updated even though the task text only named
  `generate_doc.go`).
- `helm/go.mod` / `helm/go.sum` — only if `go build`/`dagger develop` adds new
  dependencies (none expected; no new Go imports required beyond what exists).

### Files to create
- `helm/migrate.go` — new `MigrateValuesTags` function.

### Files NOT to touch
- `helm/update.go`, `helm/lint.go`, `helm/push.go`, `helm/pipeline.go` —
  unaffected (pipeline.go calls `GenerateSchema`/`GenerateDocumentation` by
  unchanged signatures).
- `helm/templates/*` — unaffected.
- `helm/dagger.json` — unaffected (no new dependencies; `git-module` stays).
- `lib/helper/command.go` — `ForgeCommand`/`ForgeCommandf`/`ForgeScript` reused
  as-is.

---

## 3. Research findings

### 3.1 Docker image (release 0.0.3)
- Image: **`ghcr.io/disaster37/readme-generator-for-helm:0.0.3`**
  - Source: https://github.com/disaster37/readme-generator-for-helm/pkgs/container/readme-generator-for-helm
  - Digest: `sha256:9db59f792ee3ab535e1dba3bfda3516697420b35d3a248defaf2e0f551b7deb1`
  - Release 0.0.3 notes: "Add sub comment to migrate values.yaml tags to new
    syntaxes." https://github.com/disaster37/readme-generator-for-helm/releases/tag/0.0.3
- Dockerfile
  (https://raw.githubusercontent.com/disaster37/readme-generator-for-helm/main/Dockerfile):
  base `alpine:3.20`, binary copied to `/usr/local/bin/readme-generator`,
  `ENTRYPOINT ["readme-generator"]`. Binary is on `PATH`.

### 3.2 Binary name & CLI flags
- Binary: `readme-generator` (same name as the old npm package's bin).
- **Breaking change vs. old Bitnami tool:** the fork uses **subcommands**:
  `readme`, `schema`, `all`, `migrate`. The old tool had no subcommand
  (`readme-generator -r README.md --values values.yaml`).
- Flags (https://github.com/disaster37/readme-generator-for-helm#usage):
  - `-v` / `--values` — path to `values.yaml` (required, all subcommands).
  - `-c` / `--config` — config JSON (optional).
  - `-r` / `--readme` — path to `README.md` (required for `readme`/`all`).
  - `-s` / `--schema` — path to `values.schema.json` (required for `schema`/`all`).
  - `--chart`, `--no-auto-deps`, `--enrich-deps` — dependency docs options.
  - `migrate`-only: `--dry-run`, `--output <path>` (default: rewrite `--values`
    in place), `--no-normalize-arrays`.
- Correct invocations for this module:
  - README: `readme-generator readme -v values.yaml -r README.md`
    (with config: `readme-generator readme -c <cfg> -v values.yaml -r README.md`)
  - Schema: `readme-generator schema -v values.yaml -s values.schema.json`
    (with config: `readme-generator schema -c <cfg> -v values.yaml -s values.schema.json`)
  - Migrate: `readme-generator migrate -v values.yaml`
    (with config: `readme-generator migrate -c <cfg> -v values.yaml`)
    (to new file: `readme-generator migrate -v values.yaml --output <path>`)

### 3.3 Annotation / tag format differences
- Tag tokens are unchanged from the Bitnami original: `@param`, `@section`,
  `@skip`, `@extra`, `@descriptionStart`/`@descriptionEnd`. Comment prefix is
  configurable (`##` by default, matching the Bitnami convention). So existing
  chart annotations remain valid; no annotation-format migration is needed.
- **The only semantic difference is array-index syntax in metadata paths.**
  The old JS tool used concrete indices (`[0]`, `[1]`); the Go fork uses generic
  `[]`. The fork's parser auto-normalizes `[N]`→`[]` at parse time, but
  recommends cleaning source files.

### 3.4 What "migrate tags on values.yaml" means
- It means running the fork's `migrate` subcommand, added specifically in
  release 0.0.3. It rewrites `[N]` → `[]` inside `@param`/`@skip`/`@extra`
  metadata-comment path tokens in `values.yaml`, leaving YAML values, modifiers,
  and descriptions byte-for-byte unchanged. It is idempotent and safe to run
  repeatedly. It does NOT touch non-comment lines.
  - Source: https://github.com/disaster37/readme-generator-for-helm#automated-migration-with-migrate
- This is a source-file rewrite of metadata comments, not a runtime data
  migration. Concretely, a line like `# @param jobs[0].name Job name` becomes
  `# @param jobs[].name Job name`.

### 3.5 Dagger `WithExec` vs. image ENTRYPOINT (critical)
- Per the Dagger Go SDK docs
  (https://pkg.go.dev/dagger.io/dagger), `Container.WithExec` **ignores the OCI
  entrypoint by default**; `ContainerWithExecOpts.UseEntrypoint` must be set
  explicitly to prepend the entrypoint.
- Therefore the existing code pattern (`helper.ForgeCommand("readme-generator
  -r ...")`, which puts `readme-generator` as the first arg) keeps working with
  the new image: the args run as the full command and the image's
  `ENTRYPOINT ["readme-generator"]` is ignored. **No `WithEntrypoint` change is
  needed.** We keep the existing convention of prefixing args with
  `readme-generator` (consistent with `lint.go` prefixing `helm ...`).

---

## 4. Detailed changes

### 4.1 `helm/main.go`
Replace the default `GeneratorContainer` branch in `New()`.

Current:
```go
if baseGeneratorContainer != nil {
    helm.GeneratorContainer = baseGeneratorContainer
} else {
    helm.GeneratorContainer = dag.Container().
        From("node:21-alpine").
        WithExec(helper.ForgeCommand("npm install -g @bitnami/readme-generator-for-helm"))
}
helm.GeneratorContainer = helm.GeneratorContainer.WithWorkdir(sourceDirectory)
```

New:
```go
if baseGeneratorContainer != nil {
    helm.GeneratorContainer = baseGeneratorContainer
} else {
    // ghcr.io/disaster37/readme-generator-for-helm:0.0.3
    // digest: sha256:9db59f792ee3ab535e1dba3bfda3516697420b35d3a248defaf2e0f551b7deb1
    helm.GeneratorContainer = dag.Container().
        From("ghcr.io/disaster37/readme-generator-for-helm:0.0.3")
}
helm.GeneratorContainer = helm.GeneratorContainer.WithWorkdir(sourceDirectory)
```

Notes:
- Remove the `npm install` step entirely (binary is baked into the image).
- Keep `baseGeneratorContainer` override parameter and its `+optional` semantics
  unchanged. `WithSource`/`WithWorkDir` behavior for `GeneratorContainer` is
  unchanged (they already mount `src` and set workdir on it).
- The `helper` import remains used (still used by other code paths in the
  module; verify `go vet` after edit — if `helper` becomes unused in main.go
  alone it is still imported elsewhere, but main.go itself still uses
  `helper.ForgeScript` in `WithRepository`, so the import stays).

### 4.2 `helm/generate_doc.go`
Update the two `WithExec` commands to use the `readme` subcommand and the `-v`
short flag for values.

Current:
```go
if configFile == "" {
    container = container.
        WithExec(helper.ForgeCommandf("readme-generator -r %s --values values.yaml", targetFile))
} else {
    container = container.
        WithExec(helper.ForgeCommandf("readme-generator -c %s -r %s --values values.yaml", configFile, targetFile))
}
```

New:
```go
if configFile == "" {
    container = container.
        WithExec(helper.ForgeCommandf("readme-generator readme -v values.yaml -r %s", targetFile))
} else {
    container = container.
        WithExec(helper.ForgeCommandf("readme-generator readme -c %s -v values.yaml -r %s", configFile, targetFile))
}
```

Everything else in the function (signature, defaults, return) unchanged.

### 4.3 `helm/generate_schema.go`
Same CLI-shape update for the `schema` subcommand.

Current:
```go
if configFile == "" {
    container = container.
        WithExec(helper.ForgeCommandf("readme-generator -s %s --values values.yaml", targetFile))
} else {
    container = container.
        WithExec(helper.ForgeCommandf("readme-generator -c %s -s %s --values values.yaml", configFile, targetFile))
}
```

New:
```go
if configFile == "" {
    container = container.
        WithExec(helper.ForgeCommandf("readme-generator schema -v values.yaml -s %s", targetFile))
} else {
    container = container.
        WithExec(helper.ForgeCommandf("readme-generator schema -c %s -v values.yaml -s %s", configFile, targetFile))
}
```

### 4.4 `helm/migrate.go` (new file)

Full proposed content:

```go
package main

import (
	"context"
	"dagger/helm/internal/dagger"

	"github.com/disaster37/dagger-library-go/lib/helper"
)

// MigrateValuesTags permit to migrate values.yaml metadata tags
// It rewrites old concrete array indices ([0], [1]) in @param / @skip / @extra
// metadata comment paths to the new generic [] syntax used by the Go
// readme-generator-for-helm fork. The operation is idempotent and only modifies
// metadata comment lines; YAML values, modifiers, and descriptions are preserved.
// It returns the migrated values file.
func (m *Helm) MigrateValuesTags(
	ctx context.Context,

	// The values file to migrate
	// +optional
	// +default="values.yaml"
	valuesFile string,

	// Config file for readme-generator
	// +optional
	configFile string,

	// Output file path. If empty, valuesFile is rewritten in place.
	// +optional
	outputFile string,
) (migratedFile *dagger.File, err error) {

	container := m.GeneratorContainer

	if valuesFile == "" {
		valuesFile = "values.yaml"
	}

	effectivePath := valuesFile

	if outputFile == "" {
		if configFile == "" {
			container = container.
				WithExec(helper.ForgeCommandf("readme-generator migrate -v %s", valuesFile))
		} else {
			container = container.
				WithExec(helper.ForgeCommandf("readme-generator migrate -c %s -v %s", configFile, valuesFile))
		}
	} else {
		effectivePath = outputFile
		if configFile == "" {
			container = container.
				WithExec(helper.ForgeCommandf("readme-generator migrate -v %s --output %s", valuesFile, outputFile))
		} else {
			container = container.
				WithExec(helper.ForgeCommandf("readme-generator migrate -c %s -v %s --output %s", configFile, valuesFile, outputFile))
		}
	}

	migratedFile = container.File(effectivePath)
	return migratedFile, nil
}
```

Design notes:
- Runs on `m.GeneratorContainer` (same container as `GenerateDocumentation` /
  `GenerateSchema`), which already has `src` mounted and workdir set via
  `WithSource`/`WithWorkDir`. No new container field needed.
- No `InsecureRootCapabilities` required: the alpine image runs as root and the
  fork performs an atomic in-place write to the mounted filesystem. (Contrast
  with `update.go` which sets `InsecureRootCapabilities: true` for `yq
  --inplace`; that flag is about Linux capabilities, not file permissions, and
  is not needed here. If write-permission errors appear in practice, add
  `dagger.ContainerWithExecOpts{InsecureRootCapabilities: true}` as a fallback,
  but do not add it preemptively.)
- `dryRun` is intentionally NOT exposed on the Dagger surface: dry-run prints to
  stdout and writes nothing, which is a terminal-preview convenience rather
  than a pipeline need. Users who want a non-destructive preview can set
  `outputFile` to a new path (e.g. `values.migrated.yaml`) and inspect that
  file. (Out of scope: a future `MigrateValuesTagsPreview` returning `string`
  stdout if a preview surface is ever wanted.)
- `ctx context.Context` is accepted for convention consistency with
  `UpdateChart`/`UpdateValues`/`Lint`, even though it is not currently used
  inside the body (the `container.File(...)` is lazy). Keep it so the signature
  matches the module's ergonomic style and remains forward-compatible.

---

## 5. Data structures & function signatures (summary)

No new struct fields. `Helm` struct unchanged. New/changed signatures:

```go
// main.go (constructor branch only; signature unchanged)
func New(src *dagger.Directory, baseHelmContainer *dagger.Container, baseGeneratorContainer *dagger.Container, baseYqContainer *dagger.Container) *Helm

// generate_doc.go (signature unchanged)
func (m *Helm) GenerateDocumentation(targetFile string, configFile string) (readmeFile *dagger.File, err error)

// generate_schema.go (signature unchanged)
func (m *Helm) GenerateSchema(targetFile string, configFile string) (schemaFile *dagger.File, err error)

// migrate.go (NEW)
func (m *Helm) MigrateValuesTags(ctx context.Context, valuesFile string, configFile string, outputFile string) (migratedFile *dagger.File, err error)
```

Dagger param annotations on `MigrateValuesTags`:
- `valuesFile`: `+optional`, `+default="values.yaml"`
- `configFile`: `+optional`
- `outputFile`: `+optional`

---

## 6. Edge cases

1. **Empty/missing `values.yaml`**: `readme-generator migrate -v values.yaml`
   will fail if the file is absent. The error surfaces as a Dagger exec error
   (non-zero exit). `pipeline.go` already guards `GenerateSchema`/
   `GenerateDocumentation` behind a `values.yaml` existence check; if
   `MigrateValuesTags` is later wired into the pipeline, add the same guard.
   For now the function assumes the caller provides a chart source containing
   `values.yaml`.
2. **Missing config file**: if `configFile` is set but absent, the fork errors
   with a non-zero exit; surfaces as a Dagger exec error. No pre-validation
   needed (consistent with `GenerateDocumentation` which also does not
   pre-validate).
3. **Image tag mutability**: `0.0.3` is a mutable tag. The rest of the module
   uses mutable tags (`alpine/helm:3.14.3`, `mikefarah/yq:4.35.2`), so this is
   consistent. For reproducibility, the digest
   `sha256:9db59f792ee3ab535e1dba3bfda3516697420b35d3a248defaf2e0f551b7deb1`
   is recorded in a code comment above the `From(...)` call; pin to the digest
   only if the project later adopts digest-pinning repo-wide.
4. **`baseGeneratorContainer` backwards compatibility**: callers that supply a
   custom container must now provide a `readme-generator` binary that supports
   the subcommand CLI (`readme`/`schema`/`migrate`). A custom container still
   holding the old npm-installed Bitnami JS tool will break because
   `generate_doc.go`/`generate_schema.go` now emit subcommands. This is a
   documented breaking change for `baseGeneratorContainer` users.
5. **Chart source not present**: `WithSource` must have been called (it is, in
   `New()`). If `Src` is empty, `GeneratorContainer` has no `values.yaml` and
   exec fails — surfaces as a Dagger error. No extra validation.
6. **Idempotency**: running `MigrateValuesTags` repeatedly is safe; the fork's
   `migrate` is idempotent. No guard against double-migration needed.
7. **`outputFile` equal to `valuesFile`**: equivalent to in-place; the fork
   handles it. No special-casing required.
8. **Nested arrays**: the fork's `migrate` rewrites all `[N]` tokens on
   metadata lines (not just the first), so nested arrays like
   `jobs[0].tasks[1].name` are fully migrated in one pass (unlike the sed
   one-liner the README warns about). No multi-pass logic needed.

---

## 7. Error handling & validation

- No new Go-level validation logic is required; the module's existing pattern
  is to let the underlying tool fail and surface the Dagger exec error
  (see `generate_doc.go`, `lint.go`). Follow that pattern.
- Do NOT add `panic` calls (the module uses `panic` only for constructor-arg
  validation in `New`/`WithRepository`, not for runtime file operations).
- Return the tool's non-zero exit as `err` from the Dagger function (Dagger
  converts exec failures to errors automatically when the file is consumed).
- `go vet` must stay clean; `ctx` is accepted but unused in the body — this is
  acceptable (golangci-lint `unused-parameter` is not enabled in this repo;
  there is no `.golangci.yml` in the repo root or `helm/`). If a linter is ever
  introduced that flags unused parameters, prefix with `_` — but do not do that
  now since it would break the Dagger function-arg convention.

---

## 8. Testing / verification steps

No `Makefile`, `.golangci.yml`, or `.markdownlint.json` exists in the repo root
or `helm/` (confirmed via glob). CI is driven by Dagger itself (see
`helm/pipeline.go` `GenerateCi`). Verification is manual + Dagger CLI:

1. **Compile** (from `helm/`):
   ```
   cd helm && go build ./...
   ```
2. **Vet**:
   ```
   cd helm && go vet ./...
   ```
3. **Dagger develop / module load**:
   ```
   dagger -m helm develop
   ```
4. **List functions** (confirm `MigrateValuesTags` appears with correct params):
   ```
   dagger -m helm functions
   ```
5. **Behavior: GenerateDocumentation** against a sample chart with `values.yaml`
   using `@param` annotations:
   ```
   dagger -m helm -s ./helm call --src=. generate-documentation --target-file=README.md
   ```
   Confirm `README.md` is produced and the Parameters table renders.
6. **Behavior: GenerateSchema**:
   ```
   dagger -m helm -s ./helm call --src=. generate-schema --target-file=values.schema.json
   ```
7. **Behavior: MigrateValuesTags** on a `values.yaml` containing old `[0]`
   indices in `@param` comments:
   ```
   dagger -m helm -s ./helm call --src=. migrate-values-tags --values-file=values.yaml
   ```
   Confirm `[0]` → `[]` in metadata comments only; YAML values unchanged.
8. **Behavior: MigrateValuesTags with outputFile** (non-destructive):
   ```
   dagger -m helm -s ./helm call --src=. migrate-values-tags --output-file=values.migrated.yaml
   ```
   Confirm `values.yaml` untouched and `values.migrated.yaml` contains the
   rewrite.
9. **Idempotency**: run step 7 twice; `git diff` should be empty after the
   second run.
10. **Custom container override**: invoke `New` with a `baseGeneratorContainer`
    built from the same `ghcr.io/...:0.0.3` image to confirm the override path
    still works (smoke test via a small Dagger call or unit if any exist — none
    found in `helm/`).

---

## 9. Project conventions observed (must follow)

- **Command forging**: use `helper.ForgeCommand` / `helper.ForgeCommandf` from
  `github.com/disaster37/dagger-library-go/lib/helper` (string split on spaces),
  not raw `[]string{...}` literals. `update.go` is the exception (uses a literal
  slice because it passes `dagger.ContainerWithExecOpts`); `generate_doc.go`,
  `generate_schema.go`, `lint.go` all use `helper.ForgeCommand*`. New
  `migrate.go` follows the `ForgeCommandf` convention.
- **Dagger param annotations**: `+required` / `+optional` / `+default=...` on
  each arg, with a preceding comment line describing the arg. First line of the
  method's doc comment is the short description shown in `dagger functions`.
- **Container reuse**: generator operations run on `m.GeneratorContainer`; yq
  operations on `m.YqContainer`; helm operations on `m.HelmContainer`. Do not
  introduce a new container field for migration.
- **`InsecureRootCapabilities`**: used in `update.go` for `yq --inplace`. Do
  NOT add it preemptively to `migrate.go` (see §4.4 notes).
- **Return types**: file-producing functions return `*dagger.File`
  (`GenerateDocumentation`, `GenerateSchema`, `UpdateChart`, `UpdateValues`).
  `MigrateValuesTags` follows this.
- **`ctx context.Context` first arg**: present on functions that may need it
  (`Lint`, `UpdateChart`, `UpdateValues`, `Ci`); included on `MigrateValuesTags`
  for consistency even though the body is lazy.

---

## 10. Risks / open items

- **Breaking change for `baseGeneratorContainer` users**: any external caller
  passing a custom container with the old Bitnami JS `readme-generator` will
  break because we now emit subcommands. Mitigation: document in the function
  doc comments / module README that the custom container must provide a
  subcommand-compatible `readme-generator` (the fork). (Out of scope: editing
  module README.)
- **Image availability**: `ghcr.io/disaster37/readme-generator-for-helm:0.0.3`
  is a single-maintainer package with 0 downloads at research time. If the
  image is unavailable, `From(...)` will fail at runtime. Mitigation: the
  `baseGeneratorContainer` override lets callers supply a mirror or a
  self-built image (the fork's Dockerfile is reproducible).
- **No automated tests in `helm/`**: there are no `*_test.go` files in the
  module, so verification is manual via `dagger call` (§8).

---

## 11. Out of scope

- Wiring `MigrateValuesTags` into `pipeline.go`'s `Ci` flow (the pipeline
  currently runs `GenerateSchema` + `GenerateDocumentation` + `Lint` + `Push`;
  migration is a one-time chart-author action, not a per-CI step).
- Exposing `--dry-run` on the Dagger surface (use `outputFile` for
  non-destructive preview instead).
- Updating `helm/README.md` or module-level docs.
- Digest-pinning the image (kept as a comment for future use).
- Adding `--chart` / `--no-auto-deps` / `--enrich-deps` options to
  `GenerateDocumentation`/`GenerateSchema` (the fork supports them, but the
  current functions do not expose them; adding them is a separate enhancement).
