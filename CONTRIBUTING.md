# Contributing

## 1. Local Workflow

Use the Nix development shell for repository work:

```sh
nix develop
```

Run the focused local checks before committing:

```sh
nix fmt
nix flake check --print-build-logs
nix build --print-build-logs
```

For a fast Go-only loop while editing viewer behavior, use:

```sh
nix develop .#ci --command go test ./...
```

For browser e2e coverage, including Mermaid SVG rendering, use:

```sh
nix develop .#ci --command go test -tags browser_e2e ./cmd/mdview
```

Stage newly added files before running the full flake check. The pre-commit
check surface is part of `nix flake check`, and staged files make the local
result match the CI workflow more closely.

## 2. Viewer Development

`mdview` is the preferred public CLI. It serves a Markdown directory through a
loopback HTTP server. Markdown files render as sanitized browser preview pages,
and Mermaid fences render in the browser from the vendored static asset.

Useful local commands:

```sh
go test ./...
nix build --print-build-logs
./result/bin/mdview --version
```

The Mermaid browser bundle is vendored at
`internal/assets/static/vendor/mermaid.min.js`. Update it with the source and
integrity procedure in `internal/assets/MERMAID.md`; normal builds and runtime
execution must remain npm-free.

Keep user install and usage procedures in `README.md`. Keep local Nix workflow
details here.

## 3. Nix And CI

The flake owns the reproducible toolchain, formatter checks, release tooling,
and CI entry points. Keep `go.mod`, `flake.nix`, and the Go override aligned
when changing Go versions.

The primary CI workflow is `.github/workflows/ci.yml` with workflow name `ci`.
It runs Nix checks, Nix build, Go vulnerability checks, and secret scanning.
`AGENTS.md` is the source of truth for agent guidance, and `CLAUDE.md` points
Claude-compatible tooling to it.

Update the pinned Go patch override with:

```sh
nix run .#update-go-toolchain
```

The scheduled Go toolchain update workflow has been retired. When that command
changes the Go override, run the required Nix and Go checks locally and open the
pull request manually.

## 4. Commit Expectations

Use conventional commit messages such as `docs: update contributor guide` or
`fix(server): reject file roots`. Keep docs, behavior changes, workflow changes,
and dependency changes separated when the review surface is meaningfully
different.

Do not include machine-local absolute paths in commits, public docs, issues, or
pull request text. Use repo-relative paths such as `README.md` and
`.github/workflows/ci.yml`.

## 5. Releases

Release work is documented in `RELEASING.md`. Run the pre-release checklist
there before creating a release tag or using the manual release workflow.
