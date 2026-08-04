# Bound product backlog

This is the working backlog for improving Bound. Items should be implemented in
small, reviewable slices and checked off only after tests and documentation are
updated.

## In progress

- [ ] Add architecture drift reporting for declared modules, source ownership,
      implementation dependencies, and stale mappings.

## Next up: developer experience

- [ ] Add `bound review --output <path>` and Markdown/SVG output options to the
      CLI instead of requiring library callers for non-HTML exports.
- [ ] Add golden tests for formatted `.bo`, compiler IR, diagnostics, Mermaid,
      and generated review HTML.

## Language server and editor

- [ ] Add workspace-aware import resolution and diagnostics across all `.bo` and
      `.bom` files.
- [ ] Add go-to-definition, find references, document symbols, hover, rename,
      formatting, and code actions to the Bound LSP.
- [ ] Add a portable published Tree-sitter grammar repository and test clean Zed
      extension installation from a fresh machine.
- [ ] Add an automated Zed extension validation job for grammar, highlights,
      language registration, icon theme, and LSP startup.

## Architecture language

- [ ] Add a documented formatter specification and preserve source comments and
      documentation locations through formatting.
- [ ] Add explicit severity configuration for architecture quality rules.
- [ ] Add richer modeling and validation for events, queues, databases, external
      systems, and polymorphism.
- [ ] Add schema migration tooling for compiler IR versions.
- [ ] Add incremental parsing and compiler caching for large repositories.

## Agent automation and CI/CD

- [ ] Make architecture-refresh PRs idempotent and avoid duplicate no-op PRs.
- [ ] Add a generated validation report and changed-facts summary to refresh PRs.
- [ ] Restrict agent file scope, diff size, permissions, and runtime explicitly.
- [ ] Add SARIF output for architecture violations.
- [ ] Publish signed CLI, LSP, skill, and Zed extension release artifacts.
- [ ] Add a release smoke test on a clean checkout and supported platforms.

## Done in the current product pass

- [x] Replace compiler language dispatch with an injectable backend registry.
- [x] Refactor production packages into `boundary`, `lib`, and `infrastructure`.
- [x] Add stable diagnostic codes to CLI, JSON, LSP-facing compiler data, and the public Go API.
- [x] Add `bound doctor` with human-readable and `--json` output.
- [x] Add `bound diff` for architecture changes between two specifications.
- [x] Add `bound init` scaffolding.
- [x] Add `bound fmt`, including CI check mode.
- [x] Add `bound check` with design-only, watch, and JSON modes.
- [x] Add headless review generation with `review --no-open`.
- [x] Validate design-first and external examples in CI.

## Architecture decisions

Implementation backends are selected through `BackendRegistry`, not a compiler
language switch. Built-in analyzers implement the common `Backend` interface;
applications can register additional languages or replace a backend in tests.
