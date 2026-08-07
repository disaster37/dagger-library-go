# Contributing

PRs are always welcome.

## Release process

This repo uses explicit semver tags (e.g. `2.0.3`) for Dagger module references — **not** branch-based versions like `@v2`.

Before creating a new release, run the release script to bump the version across all modules:

```bash
./scripts/release.sh 2.0.4
```

This updates:
- `VERSION` files in `helm/`, `golang/`, `image/`, `operator-sdk/` (used at runtime by `GenerateCi` to auto-detect the module version)
- `@<old-version>` references in all README files

After that, commit and tag:

```bash
git add -A
git commit -m "Release 2.0.4"
git tag 2.0.4
git push --follow-tags
```

### Why this matters

Dagger modules no longer resolve branch-based references (`@v2`). They require explicit version tags (`@2.0.4`). The `GenerateCi` function reads the embedded `VERSION` file to build the correct `--module-ref` default, so generated pipelines always reference the correct version for the tag they were built from.
