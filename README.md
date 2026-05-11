# markdown-remote-viewer

`markdown-remote-viewer` provides `mdview`, a small Go CLI that serves a
Markdown directory through a loopback HTTP server. It is designed as a
single-binary foundation for local and SSH-friendly Markdown viewing workflows.

Implementation status: the initial Go CLI, HTTP server, embedded static assets,
CI, and release metadata are present. Markdown files are currently served as
files; browser-side Markdown rendering and Mermaid support are tracked as
follow-up work.

## 1. Project Docs

- `CONTRIBUTING.md` covers local development workflow and repository checks.
- `RELEASING.md` covers tag-based and manual releases, version metadata, and
  expected artifacts.
- `.goreleaser.yaml` defines the platform archives for the `mdview` binary.

## 2. Install

With Go:

```sh
go install github.com/i9wa4/markdown-remote-viewer/cmd/mdview@latest
```

## 3. Upgrade

For Go installs, rerun the install command:

```sh
go install github.com/i9wa4/markdown-remote-viewer/cmd/mdview@latest
```

For a pinned version, replace `latest` with a release tag:

```sh
go install github.com/i9wa4/markdown-remote-viewer/cmd/mdview@vX.Y.Z
```

## 4. Usage

Serve the current directory:

```sh
mdview
```

Serve a specific directory:

```sh
mdview docs
```

Bind to a fixed local port:

```sh
mdview --port 8080 docs
```

Serve from an Ubuntu SSH destination and open it from a Mac on the same
Tailnet:

```sh
mdview --tailscale --port 8080 docs
```

By default, `mdview` binds to `127.0.0.1` and prints the effective local URL.
The default does not expose the server publicly.

Use `--tailscale` when the server machine and your local browser are connected
through Tailscale. In that mode, `mdview` detects the server machine's Tailnet
IPv4 address, binds there, and prints a URL that can be opened directly from
the Mac. Open the `URL:` line from the startup output. `--tailscale` is
explicit and cannot be combined with `--addr`.

| Invocation             | Purpose                                      |
| ---------------------- | -------------------------------------------- |
| `mdview`               | Serve the current directory on loopback.     |
| `mdview docs`          | Serve `docs` on loopback.                    |
| `mdview --port 8080 .` | Serve the current directory on a fixed port. |
| `mdview --tailscale .` | Serve on the Tailscale network.              |
| `mdview --version`     | Print version metadata.                      |
