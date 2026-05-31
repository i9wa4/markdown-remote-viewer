# markdown-remote-viewer

`markdown-remote-viewer` provides `mdview`, a small Go CLI that serves a
Markdown directory through a read-only loopback HTTP server. It is designed as
a single-binary foundation for local and SSH-friendly Markdown viewing
workflows.

Implementation status: the Go CLI, HTTP server, embedded static assets,
Markdown-to-HTML preview, browser-side Mermaid rendering, CI, and release
metadata are present. Mermaid uses an embedded vendored browser bundle, so
normal builds and runtime viewing do not need npm or a CDN.

## 1. Architecture

```mermaid
flowchart TD
    CLI[mdview CLI] --> Bind[Choose bind address]
    Bind -->|default| Loopback[127.0.0.1 loopback]
    Bind -->|--tailscale| Tailnet[Tailnet IPv4]
    Bind --> Server[HTTP server]
    Browser[Browser] -->|HTTP GET| Server
    Server -->|embedded assets| Assets[CSS assets]
    Server -->|root-contained paths| Root[Selected directory]
    Root -->|*.md| Markdown[Goldmark GFM render]
    Markdown --> Sanitizer[bluemonday sanitize]
    Sanitizer --> Security[Markdown CSP allows same-origin viewer scripts]
    Security --> Browser
    Root -->|other files| Static[Raw static file serving]
    Static --> Browser
    Markdown -->|mermaid fences| Mermaid[Vendored Mermaid browser render]
    Mermaid --> Browser
```

## 2. Project Docs

- `CONTRIBUTING.md` covers local development workflow and repository checks.
- `RELEASING.md` covers tag-based and manual releases, version metadata, and
  expected artifacts.
- `.goreleaser.yaml` defines the platform archives for the `mdview` binary.

## 3. Install

With Go:

```sh
go install github.com/i9wa4/markdown-remote-viewer/cmd/mdview@latest
```

From a local checkout:

```sh
go build -o mdview ./cmd/mdview
./mdview docs
```

## 4. Upgrade

For Go installs, rerun the install command:

```sh
go install github.com/i9wa4/markdown-remote-viewer/cmd/mdview@latest
```

For a pinned version, replace `latest` with a release tag:

```sh
go install github.com/i9wa4/markdown-remote-viewer/cmd/mdview@vX.Y.Z
```

## 5. Build From Source

A fresh checkout can build one runnable `mdview` binary with Go:

```sh
go build -trimpath -o ./mdview ./cmd/mdview
```

The binary includes the viewer stylesheet through Go embedding, so no runtime
asset directory is required.

## 6. Usage

The stable command surface is `mdview [FLAGS] [PATH]`. `mdview` does not use
subcommands for remote access; use flags on the same command instead.

| Flag          | Behavior                                                       |
| ------------- | -------------------------------------------------------------- |
| `--addr ADDR` | Bind to `ADDR`; defaults to `127.0.0.1`.                       |
| `--port PORT` | Bind to `PORT`; `0` asks the OS for an available port.         |
| `--open`      | Open the primary printed URL in the local browser.             |
| `--tailscale` | Bind to the detected Tailnet IPv4 address for Tailnet access.  |
| `--no-qr`     | In Tailnet mode, print URL text without terminal QR output.    |
| `--version`   | Print version metadata.                                        |
| `--help`      | Print command help.                                            |

Startup always prints the effective `URL:` line. `--no-qr` is the text-only
Tailnet startup mode; there is no separate `share` or print-only subcommand.

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

For SSH forwarding, keep the server loopback-bound on the remote machine and
forward a local port to it:

```sh
mdview --port 8080 docs
```

From your local machine:

```sh
ssh -L 8080:127.0.0.1:8080 user@example-host
```

Then open `http://127.0.0.1:8080/` locally.

Use `--tailscale` when the server machine and your local browser are connected
through Tailscale. In that mode, `mdview` detects the server machine's Tailnet
IPv4 address, binds there, and prints a URL that can be opened directly from
the Mac. Open the `URL:` line from the startup output. Interactive Tailnet
startup also prints a terminal QR code for phone access; use `--no-qr` to keep
only the text output. `--tailscale` is explicit and cannot be combined with
`--addr`.

Markdown files ending in `.md` are rendered as HTML preview pages. Raw HTML in
Markdown is not trusted: rendering uses safe defaults and sanitizes generated
HTML before it is sent to the browser. Mermaid code fences render as SVG in the
browser with the embedded Mermaid asset; when browser scripts are unavailable,
the fenced source remains visible as a code block. Other files are served as
static files from the selected directory.

The server is read-only. It does not provide upload, edit, or delete routes,
and write methods are rejected. Static and directory responses keep script
execution disabled; rendered Markdown pages allow only same-origin viewer
scripts for Mermaid rendering. The bind address controls who can reach the
viewer: the default loopback address is local to the server machine, while
`--tailscale` is intended for access from trusted devices on the same Tailnet.

File access is limited to the selected directory after resolving symlinks.
Requests that traverse outside that root, including encoded traversal paths and
symlinks that point outside the root, are rejected. Symlinks that resolve to
files still contained inside the selected directory are allowed.

| Invocation                     | Purpose                                      |
| ------------------------------ | -------------------------------------------- |
| `mdview`                       | Serve the current directory on loopback.     |
| `mdview docs`                  | Serve `docs` on loopback.                    |
| `mdview --port 8080 .`         | Serve the current directory on a fixed port. |
| `mdview --tailscale .`         | Serve on the Tailscale network.              |
| `mdview --tailscale --no-qr .` | Serve on Tailscale without QR output.        |
| `mdview --version`             | Print version metadata.                      |
