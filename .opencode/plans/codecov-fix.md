# Codecov Module Token Fix

Fix the `codecov` Dagger module, which always reports "token is not valid" even
for valid tokens. The root cause is shell-based corruption of the token in
`Upload()` (unquoted `$CODECOV_TOKEN` expansion via `sh -c`) combined with
redundant token passing (`-t` flag + env var). The fix executes the uploader
directly (no shell) and relies solely on the `CODECOV_TOKEN` environment
variable, which the uploader reads natively.

---

## 1. Goal

- Tokens with shell-special characters (`$`, `!`, `;`, `&`, `*`, spaces, etc.)
  reach the codecov uploader byte-for-byte intact.
- No public API change: `New`, `WithContainer`, `Upload` signatures and the
  exported `Container` field stay exactly as they are.
- Add the module's first automated tests (none exist today).

## 2. Root cause (verified against current code)

File: `/projects/dagger-library-go/codecov/main.go`

- **Line 94:** `cmd := []string{"/bin/codecov", "-t", "$CODECOV_TOKEN", "-v"}`
- **Line 112:** `WithExec([]string{"sh", "-c", strings.Join(cmd, " ")}).Stdout(ctx)`

This runs `sh -c "/bin/codecov -t $CODECOV_TOKEN -v ..."`. The shell expands
`$CODECOV_TOKEN` **unquoted**, so any shell-special character in the token is
interpreted (word splitting, `$VAR` re-expansion, `!` history/glob handling,
`;`/`&` command separation). The token arriving at the uploader is truncated
or mangled → Codecov rejects it → "token is not valid".

Second problem: the token is passed twice — as the `CODECOV_TOKEN` env var
(line 111, `WithSecretVariable`) and as the `-t` argv value. CLI args take
precedence over the env var, so the *corrupted* `-t` value wins over the
*correct* env var value.

Verified in Codecov's official docs (docs.codecov.com/docs/codecov-uploader
and codecov-cli docs): *"A token can be passed to the CLI via the
`$CODECOV_TOKEN` environment variable or the `-t` command line argument"* and
*"The CLI automatically searches for a `$CODECOV_TOKEN` environment variable."*
The `-t` flag is therefore unnecessary when the env var is set.

## 3. Scope

### Files to modify
- `codecov/main.go` — rewrite `Upload()` exec; harden `New()` download;
  fix copy-pasted doc comments; remove now-unused `strings` import.

### Files to create
- `codecov/internal/uploadcmd/uploadcmd.go` — pure argv builder (no Dagger
  imports), so it is unit-testable without a Dagger session (see §4.2).
- `codecov/internal/uploadcmd/uploadcmd_test.go` — table-driven unit tests.

### Files NOT to touch
- `codecov/dagger.gen.go`, `codecov/internal/dagger/**`,
  `codecov/internal/querybuilder/**`, `codecov/internal/telemetry/**` —
  linguist-generated (see `codecov/.gitattributes`). Do NOT run
  `dagger develop` to regenerate: `dagger.json` pins `engineVersion: v0.16.1`
  while `go.mod` uses SDK `dagger.io/dagger v0.18.14`; regenerating with a
  mismatched CLI risks churn in `go.mod`/`dagger.json`. Since no function
  signature changes, regeneration is unnecessary. SourceMap line numbers baked
  into `dagger.gen.go` become slightly stale — cosmetic only.
- `codecov/go.mod`, `codecov/go.sum`, `codecov/dagger.json` — no new
  dependencies (tests use stdlib `testing` only).
- All other modules (`golang/`, `helm/`, `git-module/`, `lib/`, ...) — the
  codecov module is not referenced by any other `dagger.json` or Go code in
  this repo (verified by grep).

---

## 4. Research findings that shape the design

### 4.1 The uploader binary
`New()` downloads `https://uploader.codecov.io/{latest|v<version>}/linux/codecov`
to `/bin/codecov`. The reported symptom (an upload attempt reaching Codecov's
API and failing on the token) proves the current flag-style invocation
(no subcommand) is understood by the served binary, i.e. it behaves like the
legacy uploader. Both the legacy uploader and the newer codecov-cli read
`CODECOV_TOKEN` from the environment, so the fix is correct for either.
Note: the binary uploader is deprecated upstream in favor of codecov-cli
(which uses subcommands like `upload-process`). Migrating to codecov-cli is
OUT OF SCOPE (§13); this fix only repairs token passing for the current
binary.

### 4.2 Unit tests cannot live in `package main`
`codecov/internal/dagger/dagger.gen.go` has a package `init()` (lines
7904–7910) that calls `getClientParams()`, which **panics unless
`DAGGER_SESSION_PORT` and `DAGGER_SESSION_TOKEN` are set** (lines 7921–7934).
Any test binary linking `package main` (which imports that package) would
panic at startup on a plain `go test`. Therefore the testable logic must live
in a sub-package that does not import the generated SDK:
`codecov/internal/uploadcmd`. Plain `go test ./internal/uploadcmd` then runs
with no Dagger engine. This also explains why no Dagger module in this repo
has tests today (only the pure-Go `lib/pipeline` does).

### 4.3 `New()` silently installs garbage on download failure
`curl -o /bin/codecov -s <url>` (line 57) lacks `-f/--fail`: a 404/403 (e.g.
a typo'd `version`) writes the HTTP error body to `/bin/codecov`, `chmod +x`
succeeds, and the failure surfaces much later as a cryptic error. Related to
the requirement "verify the uploader binary is the right version and works".

### 4.4 Repo conventions
`lib/helper/command.go` provides `ForgeCommand` (space-split — unsafe for args
containing spaces) and `ForgeScript` (`sh -c`). Do NOT use either here: the
whole point of this fix is to avoid shell joins. Other modules (e.g.
`golang/main.go` line 88) already call `WithExec([]string{...})` directly —
direct exec is the established, correct pattern.

---

## 5. Design decisions

| # | Decision | Rationale |
|---|----------|-----------|
| D1 | Drop the `-t $CODECOV_TOKEN` argument entirely | Uploader reads `CODECOV_TOKEN` env var natively (§2). Removes the corrupted-argv-wins problem. |
| D2 | Execute `/bin/codecov` directly via `WithExec(cmd)` — no `sh -c`, no `strings.Join` | Exec-form passes each argv element verbatim; no shell interpretation, no injection surface. Token also never appears in argv/process lists/query logs (secret env vars are redacted by Dagger). |
| D3 | Keep `WithSecretVariable("CODECOV_TOKEN", token)` | Sole token channel; required by the uploader. |
| D4 | Extract argv assembly into `internal/uploadcmd.Build()` | Makes the logic unit-testable without a Dagger session (§4.2). |
| D5 | Add `-fL` to the `curl` in `New()` | Fail fast on HTTP errors instead of installing an error-page as the "uploader" (§4.3). |
| D6 | Replace the debug `ls -lah /bin/codecov` exec with `/bin/codecov --version` | Build-time proof the downloaded binary is executable and shows which version was installed. Both legacy uploader and codecov-cli support `--version`. Fallback: if any doubt arises during validation, revert this one line to the original `ls -lah` — everything else in the plan is unaffected. |
| D7 | Do not regenerate `dagger.gen.go` | Signatures unchanged; avoids engine/SDK version drift (§3). |

## 6. Implementation tasks (ordered)

### Task 1 — Create `codecov/internal/uploadcmd/uploadcmd.go`

```go
// Package uploadcmd assembles the codecov uploader command line.
// It deliberately has no Dagger dependency so it can be unit-tested
// without a Dagger session.
package uploadcmd

// Build returns the argv for the codecov uploader.
//
// The token must NEVER be part of argv: it is provided to the container as
// the CODECOV_TOKEN secret environment variable, which the uploader reads
// automatically. Embedding it in argv would expose it in process listings
// and Dagger logs, and historically it was corrupted by shell expansion.
func Build(name string, files []string, flags []string) []string {
	cmd := []string{"/bin/codecov", "-v"}

	if name != "" {
		cmd = append(cmd, "-n", name)
	}

	if len(files) > 0 {
		cmd = append(cmd, "-f")
		cmd = append(cmd, files...)
	}

	// Raw passthrough: each element becomes exactly one argv entry.
	if len(flags) > 0 {
		cmd = append(cmd, flags...)
	}

	return cmd
}
```

### Task 2 — Rewrite `Upload()` in `codecov/main.go`

Replace lines 94–112 with:

```go
	cmd := uploadcmd.Build(name, files, flags)

	return h.Container.
		WithDirectory("/project", src).
		WithSecretVariable("CODECOV_TOKEN", token).
		WithExec(cmd).Stdout(ctx)
```

- Add import `"dagger/codecov/internal/uploadcmd"`.
- **Remove the `"strings"` import** (line 21) — `strings.Join` was its only
  use; leaving it in breaks the build (`imported and not used`).
- Keep `context` and `fmt` (fmt is still used in `New`).
- The `Upload` signature, parameter comments, and `+optional` annotations
  stay byte-for-byte identical.
- Optionally add a short doc comment on `Upload`, e.g.:
  `// Upload uploads coverage reports to Codecov. The token is passed only via
  the CODECOV_TOKEN secret environment variable; the uploader binary is
  executed directly so argument values reach it unmodified.`

### Task 3 — Harden `New()` in `codecov/main.go`

- Line 57: `WithExec([]string{"curl", "-o", "/bin/codecov", "-s", urlCodecov})`
  → `WithExec([]string{"curl", "-fL", "-o", "/bin/codecov", "-s", urlCodecov})`
  (`-f` fails on HTTP >= 400; `-L` follows redirects).
- Line 59: `WithExec([]string{"ls", "-lah", "/bin/codecov"})`
  → `WithExec([]string{"/bin/codecov", "--version"})` (decision D6).
- Fix the copy-pasted doc comments (they describe the golang module):
  - Line 29 `// New initializes the golang dagger module` →
    `// New initializes the codecov dagger module`
  - Lines 35–36 `// The golang version to use when no go.mod` →
    `// The codecov uploader version to download (default: latest)`
  Note: descriptions baked into `dagger.gen.go` stay stale until a future
  regeneration — accepted (D7).

### Task 4 — Create `codecov/internal/uploadcmd/uploadcmd_test.go`

Table-driven tests (stdlib only). Required cases:

1. No optionals → `["/bin/codecov", "-v"]` exactly.
2. `name="unit"` → `["/bin/codecov", "-v", "-n", "unit"]`.
3. `name` containing a space (`"my upload"`) stays ONE argv element.
4. `files=["a.out","b.out"]` → `["/bin/codecov", "-v", "-f", "a.out", "b.out"]`
   (order preserved, single leading `-f` — existing behavior, unchanged).
5. `flags=["-F","unit","--debug"]` appended verbatim at the end.
6. A flag containing a space (`"--foo=bar baz"`) stays ONE argv element
   (regression guard vs. the old `strings.Join` + shell word-splitting).
7. All three optionals combined — full expected argv, exact order.
8. Token-leak guard: for a fully-populated call, assert no argv element is
   `-t` and none contains `CODECOV_TOKEN` or `$` (regression guard for this
   bug).

Compare with `reflect.DeepEqual`. Use `t.Run` subtests, mirroring the plain
style of `lib/pipeline/command_test.go`.

### Task 5 — Validate (see §10)

---

## 7. Edge cases

| Case | Behavior after fix |
|------|--------------------|
| `name`, `files`, `flags` all empty | Runs `/bin/codecov -v`; token via env. Same as before minus the broken `-t`. |
| Token with `$ ! ; & * ( ) spaces backticks` | Reaches uploader intact: no shell involved; env var value is passed verbatim by the container runtime. |
| Empty token secret | Uploader exits non-zero ("token not set"/similar); error propagates from `.Stdout(ctx)`. Same class of behavior as before. |
| `flags` elements with spaces | Each element is one argv entry (exec form). Previously `sh -c` + `strings.Join` word-split them. Strictly more correct; see migration notes. |
| Caller passes own token via `flags: ["-t","xxx"]` | Still works (appended to argv), but now that token is visible in the process argv — recommend the `token` parameter instead. |
| Multi-argument commands without `sh -c` | `WithExec([]string{...})` is exec-form: element 0 is the binary, rest are argv. No quoting needed or possible — this is why the shell is removed, not replaced. |
| `WithContainer()` used with a custom container | Contract unchanged: the replacement container must contain the uploader at `/bin/codecov` (same as the `base` param of `New`). `Upload` still mounts `src` at `/project` regardless of the container's workdir. `WithContainer` mutates `h.Container` in place and returns `h` — no new side effects. |
| `files` with spaces in paths | Now passed intact (previously word-split). Improvement. |
| Invalid `version` in `New()` | `curl -fL` fails the container build immediately with a clear curl error, instead of installing an HTML error page as the binary. |

## 8. Error handling

- No new error paths are introduced; do not add error swallowing.
- `WithExec(cmd).Stdout(ctx)` evaluates lazily: any failure (non-zero exit of
  the uploader — invalid token, no coverage reports found, network failure to
  codecov.io — or missing binary) surfaces as the `error` returned by
  `Upload`, with the engine including the exec's stderr in the error message.
  This is unchanged from today.
- `New()` container-build failures (e.g. curl 404 with `-fL`) surface when
  the container is first evaluated (at `Upload` time), as before.
- The only compile-time hazard: forgetting to remove the `strings` import
  (Task 2). `go build` catches it.

## 9. Testing strategy

There are currently NO tests in any Dagger module of this repo. Strategy:

1. **Unit tests (required, CI-friendly):** Task 4 tests
   `uploadcmd.Build` with plain `go test` — no Dagger engine needed
   (§4.2). This locks in argv shape, ordering, single-argument integrity,
   and the no-token-in-argv invariant.
2. **Static checks (required):** `go build ./...`, `go vet ./...`,
   `gofmt -l .` (must output nothing) inside `codecov/`.
3. **Integration smoke (optional; needs Dagger CLI + Docker):**
   a. Confirm the installed uploader runs: evaluate the module's container
      and execute `/bin/codecov --version` (e.g. through the exported
      `Container` field via `dagger call`, or a scratch Go program run under
      `dagger run`). Expect a version line and exit 0.
   b. **Token-integrity proof** (the definitive end-to-end check): build a
      fake base container whose `/bin/codecov` is a script that prints
      `sha256($CODECOV_TOKEN)` (via `WithNewFile` + chmod 0755), pass it as
      `base` to `New`, then call `Upload` with a token such as
      `upload-token-with$pecial!chars; & *(spaces) inside`. Assert the
      printed hash equals the sha256 of the original token string. Any shell
      mangling changes the hash.
   c. Real upload (only with a genuine test repo + token): run `Upload`
      against a directory containing a coverage file and confirm Codecov
      accepts it.
   Skip 3a–3c silently in environments without a Dagger engine; they are not
   prerequisites for merging the fix.

## 10. Validation checklist

Run from `/projects/dagger-library-go/codecov`:

1. `go build ./...` — succeeds (also proves `strings` import removed).
2. `go vet ./...` — clean.
3. `gofmt -l .` — no output.
4. `go test ./internal/uploadcmd/...` — all tests pass, **without** any
   `DAGGER_SESSION_*` env vars set (proves independence from the engine).
5. Grep-verify the fix: `grep -n "sh -c\|strings.Join\|-t" main.go` in
   `codecov/` returns nothing related to exec/token.
6. Confirm `codecov/dagger.gen.go`, `go.mod`, `go.sum`, `dagger.json` are
   unmodified (`git status`).
7. Optional (Dagger available): §9.3a–3c.

## 11. Migration / compatibility notes

- **Public API is unchanged**: `New(ctx, base, version) (*Codecov, error)`,
  `WithContainer(ctn) *Codecov`,
  `Upload(ctx, src, token, name, files, flags) (string, error)`, exported
  field `Container`. No caller changes required; no version bump needed.
- **Behavior deltas (all improvements, document in PR):**
  - Token is no longer placed on the command line; env var only. Anyone
    relying on argv sniffing of the token (nobody should) is affected.
  - `name`/`files`/`flags` values containing spaces or shell metacharacters
    are now passed as single, literal arguments instead of being word-split
    or interpreted by `sh`. Values that were previously broken now work;
    no previously-working value changes meaning.
  - With a conflicting `-t` in `flags` plus the env var, precedence follows
    the uploader's own rules (CLI flag wins) — same as before, but the flag
    value is no longer shell-corrupted.
- No dependency, `dagger.json`, or generated-code changes.

## 12. Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| `uploader.codecov.io/latest` someday serves codecov-cli, which requires a subcommand (`upload-process`) and would break `Upload` regardless of this fix | Low/medium, upstream-driven | The `--version` exec (D6) logs binary identity at build time; CLI migration is tracked separately (§13). |
| Stale SourceMap line numbers / doc descriptions in `dagger.gen.go` after editing `main.go` | Certain, cosmetic | Accepted (D7); fixed automatically by a future regeneration with a matching engine. |
| `--version` unsupported by some pinned uploader version | Low (both known binaries support it) | Fallback in D6: revert that single line to `ls -lah`. |
| Running `dagger develop` "to be safe" introduces engine/SDK drift | Medium if done | Explicitly forbidden by this plan (§3). |

## 13. Observations explicitly OUT of scope

- Migrating from the deprecated binary uploader to `codecov-cli`
  (`cli.codecov.io`, subcommand-based: `upload-process`), including its
  integrity-checking (SHA256SUM/GPG) recommendations.
- Whether `-f file1 file2` (one `-f`, many files) is accepted by the
  uploader vs. repeated `-f` flags — pre-existing behavior of the `files`
  parameter, unchanged by this fix.
- The same unquoted-expansion pattern exists in `helm/main.go`
  (`ForgeScript` with `${REGISTRY_USERNAME_*}`) — separate module, separate
  fix.
- Adding the optional integration tests (§9.3) as permanent `go test`
  targets gated on a Dagger session.
