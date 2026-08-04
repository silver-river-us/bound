# Bound language server in Zed

Bound includes a stdio Language Server Protocol server for `.bo` files. It
provides live parse/model diagnostics and keyword completion while editing.
Implementation backends are not run by the editor, so design documents can be
validated before their source tree exists.

## Install from a checkout

From the Bound repository:

```sh
./bound-lsp
```

The launcher builds and caches the server at `target/go/bound-lsp`. For a
stable absolute path suitable for editor configuration, build it once:

```sh
mkdir -p "$HOME/.local/bin"
go build -trimpath -o "$HOME/.local/bin/bound-lsp" ./src/boundary/lsp-server
```

Alternatively, use the repository launcher at an absolute path, for example
`/Users/you/Develop/bound/bound-lsp`.

## Install the local Zed extension

Zed does not register a new language server from `settings.json` alone. Install
the included dev extension instead:

1. Open Zed's command palette with `Cmd+Shift+P`.
2. Run `zed: install dev extension`.
3. Select the repository directory:

   ```text
   /absolute/path/to/bound/integrations/zed-extension
   ```

The extension registers the `Bound` language for `.bo` files, loads the Bound
Tree-sitter grammar and highlighting queries, and starts `bound-lsp` for live
diagnostics and completion. The current local manifest uses a `file://` grammar
repository, so this dev-extension flow is intentionally local to this checkout.

The server communicates over stdin/stdout using standard LSP framing. Do not
wrap it in a command that prints logs to stdout; diagnostics and logs belong on
stderr.
