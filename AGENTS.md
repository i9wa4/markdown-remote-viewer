# Agent Instructions

## 1. Project Scope

This repository is a Go Markdown remote viewer project. Keep changes focused on
`mdview`, its HTTP server, embedded assets, Nix workflow, GitHub Actions, and
concise repository documentation.

`CLAUDE.md` is a regular file whose body is exactly `@AGENTS.md`, so this file
is the single source of truth for agent guidance.

## 2. Development Rules

- Prefer existing Go, Nix, and GitHub Actions patterns already in the repo.
- Keep public docs and commit messages repo-relative; do not write
  machine-local absolute paths there.
- Keep user-facing install and usage procedures in `README.md` free of Nix
  commands; put Nix developer workflow details in `CONTRIBUTING.md`.
- Stage newly added files before running full Nix checks.
- Run focused checks for the files touched, then run the broader checks when a
  change affects shared server behavior, release configuration, or workflow
  contracts.
- Do not push without an explicit instruction.

## 3. Viewer Rules

`mdview` is the preferred user-facing CLI. It serves a Markdown directory over a
loopback HTTP server. Markdown rendering is implemented as a sanitized preview;
Mermaid rendering remains follow-up work, so do not document Mermaid as
implemented until the code supports it.

## 4. Required Checks

For docs-only changes, run at least:

```sh
nix fmt
```

For Go, Nix, CI, release, or shared viewer behavior changes, add:

```sh
nix develop .#ci --command go test ./...
nix flake check --print-build-logs
nix build --print-build-logs
```

For GoReleaser changes, also run:

```sh
nix develop .#cd --command goreleaser check
```
