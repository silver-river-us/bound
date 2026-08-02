# Architecture refresh agent

This repository treats `.bo` and `.bom` files as architecture contracts. An
automated refresh must be conservative:

1. Inspect the current source tree and existing architecture files before editing.
2. Update only facts supported by the checked-out source: ownership, modules,
   contracts, relationships, and implementation locators.
3. Preserve intentional boundaries and documentation. Do not redesign the
   system, rename domains, or edit generated HTML as a substitute for the
   source specification.
4. Run `go test ./...`, `go vet ./...`, and compile every architecture that has
   an available implementation root. For external-repository examples, at
   minimum parse and validate the `.bo`/`.bom` model with implementation checks
   skipped.
5. Keep the diff small and explain uncertain observations in the pull request.

The workflow creates a pull request for review. Never merge, force-push, or
modify secrets, workflow permissions, or unrelated application code.
