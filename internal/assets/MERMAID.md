# Vendored Mermaid Asset

`mdview` renders Mermaid fences in the browser with a vendored Mermaid bundle.
Normal builds and runtime execution do not use npm or a CDN.

## Current Asset

- Package: `mermaid`
- Version: `11.15.0`
- License: MIT
- Source tarball: `https://registry.npmjs.org/mermaid/-/mermaid-11.15.0.tgz`
- Tarball integrity:
  `sha512-pTMbcf3rWdtLiYGpmoTjHEpeY8seiy6sR+9nD7LOs8KfUbHE4lOUAprTRqRAcWSQ6MQpdX+YEsxShtGsINtPtw==`
- Vendored file: `internal/assets/static/vendor/mermaid.min.js`
- Vendored license: `internal/assets/static/vendor/mermaid.LICENSE`
- Vendored file SHA-256:
  `51e439abf12e72752be1b0d67ed80195fb8cd85f2c7af017d57c5c6b717b58d5`

## Update Process

1. Pick the target `mermaid` version from the npm package metadata.
2. Verify the package license and tarball integrity:

   ```sh
   npm view mermaid@<version> version license dist.tarball dist.integrity --json
   ```

3. Replace the vendored bundle:

   ```sh
   tmpdir="$(mktemp -d)"
   npm pack "mermaid@<version>" --pack-destination "$tmpdir"
   tar -xOf "$tmpdir/mermaid-<version>.tgz" package/dist/mermaid.min.js \
     > internal/assets/static/vendor/mermaid.min.js
   tar -xOf "$tmpdir/mermaid-<version>.tgz" package/LICENSE \
     > internal/assets/static/vendor/mermaid.LICENSE
   sha256sum internal/assets/static/vendor/mermaid.min.js
   ```

4. Update this file with the new version, tarball integrity, and file SHA-256.
5. Run the Go and browser e2e checks documented in `CONTRIBUTING.md`.
