# markdown-remote-viewer

`markdown-remote-viewer` provides `markserve`, a small Go CLI that serves a
Markdown directory through a loopback HTTP server. It is designed as a
single-binary foundation for local and SSH-friendly Markdown viewing workflows.

Implementation status: the initial Go CLI, HTTP server, embedded static assets,
Nix development environment, CI, and release metadata are present. Markdown
files are currently served as files; browser-side Markdown rendering and
Mermaid support are tracked as follow-up work.

## 1. Project Docs

- `RELEASING.md` covers tag-based and manual releases, GoReleaser, Nix checks,
  version metadata, and expected artifacts.
- `.goreleaser.yaml` defines the platform archives for the `markserve` binary.

## 2. Install

From this repository:

```sh
nix build
./result/bin/markserve --version
```

With Go:

```sh
go install github.com/i9wa4/markdown-remote-viewer/cmd/markserve@latest
```

## 3. Upgrade

For Go installs, rerun the install command:

```sh
go install github.com/i9wa4/markdown-remote-viewer/cmd/markserve@latest
```

For a pinned version, replace `latest` with a release tag:

```sh
go install github.com/i9wa4/markdown-remote-viewer/cmd/markserve@vX.Y.Z
```

For Nix builds from a checkout, update the checkout and rebuild:

```sh
git pull --ff-only
nix build
```

## 4. Usage

Serve the current directory:

```sh
markserve
```

Serve a specific directory:

```sh
markserve docs
```

Bind to a fixed local port:

```sh
markserve --port 8080 docs
```

By default, `markserve` binds to `127.0.0.1` and prints the effective local URL.
The default does not expose the server publicly.

| Invocation                | Purpose                                      |
| ------------------------- | -------------------------------------------- |
| `markserve`               | Serve the current directory on loopback.     |
| `markserve docs`          | Serve `docs` on loopback.                    |
| `markserve --port 8080 .` | Serve the current directory on a fixed port. |
| `markserve --version`     | Print version metadata.                      |

## 5. Release Build

Build the release binary locally with Nix:

```sh
nix build --print-build-logs
```

Create local snapshot release archives with GoReleaser:

```sh
nix develop .#cd --command goreleaser release --snapshot --clean
```

Release publication is handled by `.github/workflows/release.yml`; see
`RELEASING.md` for the full checklist.

## 6. Development

```sh
nix develop
nix fmt
nix flake check --print-build-logs
nix build --print-build-logs
```

The flake exposes Go, workflow, Nix, Markdown, YAML, and security checks where
they are useful for this project.
