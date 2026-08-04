# Bound repository architecture

Bound follows a three-part architecture for its own implementation:

```text
src/
  boundary/
    cli/          `bound` command entrypoint
    lsp/          LSP protocol/application boundary
    lsp-server/   `bound-lsp` stdio executable
    render/       HTML, Mermaid, SVG, and Markdown output adapters
  lib/
    model/        language-neutral architecture model and validation
    parser/       `.bo` and `.bom` parsing
    compiler/     compilation orchestration and backend registry
    format/       canonical source formatting
    framework/    reusable architecture integration helpers
  infrastructure/
    analyze/      Go, Ruby, and Python implementation analyzers
    tree_sitter/  Tree-sitter support headers
```

## Dependency direction

- `boundary` may depend on `lib` and `infrastructure` adapters.
- `lib` owns the architecture language and compiler contracts. It should not
  depend on CLI, editor, or rendering details.
- `infrastructure` implements external concerns behind interfaces defined by the
  library, such as implementation-language analysis.
- `model` is the center of the language-neutral architecture model. It must not
  depend on a concrete implementation analyzer.

The compiler selects implementation analyzers through `BackendRegistry`; it does
not switch on language names. This allows an application or test to register a
new backend without modifying compiler orchestration.

## Executables

The root launchers preserve the qfile-style workflow:

```sh
./bound ...
./bound-lsp ...
```

They build the boundary packages into `target/go/`:

- `./src/boundary/cli`
- `./src/boundary/lsp-server`

The root `main.go` remains the public Go library package. It is intentionally
separate from the CLI boundary so library consumers do not import executable
code.

## Tests

Tests remain in the sibling `tests/` tree and mirror the implementation layers:

- `tests/model`
- `tests/parser`
- `tests/compiler`
- `tests/analyze`
- `tests/lsp`
- `tests/render`
- `tests/format`

New packages should keep production code under `src/` and tests under `tests/`.
