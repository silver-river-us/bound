---
name: bound-architecture-specs
description: Create or reverse-engineer Bound architecture specifications in .bo and .bom files. Use when documenting existing software from source, designing new software before implementation, defining contexts/contracts/modules/dependencies, mapping source ownership, or reviewing whether an architecture matches its implementation.
---

# Bound Architecture Specs

Create a precise, implementation-checkable architecture specification for Bound. Treat the `.bo` model as both a communication artifact and a contract: it should explain ownership, public interfaces, allowed dependencies, and implementation mapping without reproducing application code.

Read [spec-writing.md](references/spec-writing.md) before drafting or materially revising a Bound specification.

## Workflow

1. Identify the mode:
   - Existing software: recover the current architecture from source, imports, entrypoints, tests, and runtime boundaries. Mark uncertainty instead of inventing intent.
   - New software: design the desired architecture from capabilities, actors, external systems, data ownership, and use cases. Treat implementation mappings as planned constraints.
2. Establish the architecture boundary and implementation target. Use one architecture-level `implementation` declaration and, for source-backed systems, an architecture-level `entrypoint` where appropriate.
3. Define contexts around ownership and change boundaries. Give each context a small set of interfaces and expose only contracts that another context needs.
4. Define contracts before modules. Model meaningful operations and data types; keep signatures language-neutral and avoid leaking framework or package details.
5. Add modules beneath their owning contexts. Use `implements` for contract realizations and `uses` for intentional dependencies. Prefer direct, explicit dependencies over transitive assumptions.
6. Declare context relationships for every cross-context contract or type flow. A dependency is not valid merely because the code currently imports it.
7. Map source files and entrypoints. For existing systems, use `.bom` when the physical source layout cannot yet follow Bound conventions; for new systems, prefer conventional module folders and generated mappings.
8. Validate and review:
   - Run `bound compile path/to/architecture.bo` to check the model and implementation and inspect the resolved IR.
   - Run `bound review path/to/architecture.bo` to inspect the Mermaid architecture views.
   - Resolve every undeclared import, unmapped file, missing relationship, unknown contract type, and ambiguous ownership before calling the specification complete.

## Modeling rules

- Separate architecture facts from implementation facts. A context owns a business capability or cohesive responsibility; a Go package or directory is evidence, not automatically a context.
- Keep contracts narrow. Expose only interfaces that consumers need, and put contract data types inside the interface that owns their meaning.
- Use relations to express permitted cross-context collaboration, not every call. Use module `uses` declarations to express the concrete dependency edges that must be checked.
- Prefer one owner per concept. If two contexts need the same data, decide whether it is a shared value, a published contract type, or two independently owned representations.
- Make names stable and descriptive. Names should communicate architectural responsibility and survive refactors of frameworks or package names.
- Preserve uncertainty in documentation or a TODO outside the model. Do not encode guesses as relationships or contracts merely to make validation pass.
- Keep the specification honest: every declared source mapping should exist, every implementation file inside owned module folders should be mapped, and every cross-context implementation import should have a matching declaration.
- Use quality policies only when they express an agreed architectural constraint. Do not add arbitrary limits to compensate for an unclear design.

## Deliverable checklist

- Architecture has a meaningful name and implementation locator.
- Contexts have clear ownership boundaries and no accidental “miscellaneous” bucket.
- Public interfaces describe stable capabilities, operations, and data types.
- Modules are nested under contexts, implement the right contracts, and declare direct uses.
- Cross-context relations name the exposed interface when collaboration is contract-based.
- Source files and entrypoints have one clear owner.
- Existing-system specifications distinguish observed behavior from proposed target architecture.
- `bound compile` passes, and the generated review is understandable without opening the source code.
