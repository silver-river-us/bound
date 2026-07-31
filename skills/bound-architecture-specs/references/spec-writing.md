# Bound Specification Writing Reference

Use this reference when drafting a new `.bo` architecture or recovering one from an existing codebase.

## Contents

- [Choose the specification mode](#choose-the-specification-mode)
- [Investigation workflow](#investigation-workflow)
- [Bound building blocks](#bound-building-blocks)
- [Existing software template](#existing-software-template)
- [Greenfield template](#greenfield-template)
- [Dependency and ownership rules](#dependency-and-ownership-rules)
- [Validation and review](#validation-and-review)

## Choose the specification mode

### Existing software

Describe what the system actually does and how it is currently divided. Begin with entrypoints, package/module boundaries, imports, public interfaces, persistence, and external-system adapters. Use source evidence in this order:

1. Runtime entrypoints and commands.
2. Package or module dependency graph.
3. Public types, handlers, interfaces, and service boundaries.
4. Tests and fixtures, which often reveal intended behavior.
5. Configuration, databases, queues, HTTP clients, and other external boundaries.

Separate observed facts from interpretation. If the code violates the desired architecture, encode the intended target only when the user wants a target-state specification; otherwise encode the current state and let `bound compile` report the violations.

### New software

Start from capabilities and decisions rather than folders. Identify actors, use cases, external systems, data ownership, invariants, and contracts. Choose contexts around independent language, data, or deployment change. Add implementation mappings only after the conceptual boundaries are clear.

For greenfield work, an incomplete but honest specification is better than a complete diagram made of guessed dependencies. Add a contract when a boundary needs a stable agreement, not merely because two functions call each other.

## Investigation workflow

For an existing repository:

1. Locate repository and local agent instructions.
2. Find executable entrypoints and build manifests.
3. Produce a provisional package/import graph using the repository's native tooling.
4. Group packages by responsibility, data ownership, and change reason—not by directory depth alone.
5. Identify public contracts and the modules that implement them.
6. Identify cross-boundary calls and map each to a relation and `uses` declaration.
7. Map every source file in owned modules; use explicit `.bom` mappings when convention-based paths are not yet true.
8. Write the `.bo`, run `bound compile`, and use the first failure to refine the model.
9. Run `bound review` and compare the diagrams against the source and the intended audience's questions.

Keep a short evidence log while investigating: observed entrypoints, package owners, external dependencies, uncertain boundaries, and proposed changes. Do not put that log into the architecture model unless it is a stable description or relationship.

## Bound building blocks

### Architecture

Declare one implementation target and optionally a root entrypoint:

```bo
architecture Example do
  implementation go "."
  entrypoint :main
end
```

The implementation locator is relative to the `.bo` file when compiled. The model is language-neutral, but the compiler backend checks the selected implementation language.

### Contexts and interfaces

Contexts own capabilities. Interfaces are the contracts a context exposes:

```bo
context Reporting do
  interface Reports do
    value Snapshot do
      state :created_at :timestamp
    end

    behavior render(snapshot Snapshot) returns Snapshot
  end
  exposes Reports
end
```

Use `entity` for identity/lifecycle concepts and `value` for content-defined concepts. Keep operations explicit and signatures language-neutral. Contract types belong to their interface and can refer to local types or permitted qualified types.

Large contracts can be kept in `.bo` fragments and imported by name:

```bo
context Reporting do
  import Reports from "contracts/reports.bo"
  exposes Reports
end
```

The imported symbol must match the interface defined in the fragment.

### Modules

Modules describe private implementation ownership inside a context:

```bo
context Reporting do
  interface Reports do
    behavior render() returns string
  end
  exposes Reports

  module Reporting do
    module Service do
      implements Reports
      uses Storage
      files [:service]
    end
    module Storage do
      files [:store]
    end
  end
end
```

Module names conventionally derive folders (`Reporting.Service` becomes `reporting/service`). `implements` connects a module to an interface. `uses` can name a same-context module or interface, or a qualified interface in another context when a matching context relation exists.

### Relations

Declare cross-context collaboration at the context level:

```bo
Reporting -> Billing via Payments
```

The target context must expose the named interface. A relation authorizes the contract-level dependency; the consumer module should still declare the concrete `uses` dependency. Do not add relations for standard-library or third-party imports.

### Source mappings

Prefer conventional `files` declarations for new systems:

```bo
module Service do
  files [:service, :types]
  entrypoint cli
end
```

When existing paths cannot follow conventions, use an imported `.bom` map:

```bom
map Example do
  "cmd/server/main.go" -> App.Server
  "internal/report/report.go" -> Reporting.Service
end
```

Use explicit mappings as a migration tool, not as a way to avoid deciding ownership. Every implementation source file inside an owned module tree must have one mapping, and each entrypoint must have exactly one mapping.

## Existing software template

Use this as a starting point, replacing every placeholder with observed facts:

```bo
"""
Current-state architecture recovered from <repository>.
Observed on <date or revision>.
"""
architecture ExistingSystem do
  implementation go "."
  entrypoint :main

  context Boundary do
    interface Commands do
      behavior run() returns string
    end
    exposes Commands
    module Boundary do
      module CLI do
        implements Commands
        uses Application.Reporting
        files [:cli]
      end
    end
  end

  context Application do
    interface Reporting do
      behavior generate() returns string
    end
    exposes Reporting
    module Application do
      module Reporting do
        implements Reporting
        uses Infrastructure.Store
        files [:service]
      end
    end
  end

  context Infrastructure do
    interface Store do
      behavior save() returns string
    end
    exposes Store
    module Infrastructure do
      module Store do
        implements Store
        files [:store]
      end
    end
  end

  Boundary -> Application via Reporting
  Application -> Infrastructure via Store
end
```

Before finalizing, compare every declaration with the real import graph and change the model if the source disagrees. If the source is intentionally out of compliance, document that separately and keep the target-state change explicit.

## Greenfield template

For a new system, begin smaller and expand only where a real boundary exists:

```bo
architecture NewProduct do
  implementation go "."
  entrypoint :main

  context Orders do
    interface OrderManagement do
      entity Order do
        state :id :string
      end
      behavior place(order Order) returns Order
    end
    exposes OrderManagement

    module Orders do
      module Application do
        implements OrderManagement
        files [:service]
      end
    end
  end
end
```

Add a separate context when ownership, data lifecycle, deployment, or change cadence is genuinely independent. Add an interface when the boundary needs a stable contract. Add a relation only when one context intentionally collaborates with another.

## Dependency and ownership rules

Use this table while reconciling a specification with source:

| Observation | Bound expression |
| --- | --- |
| Module implements a public capability | `implements InterfaceName` |
| Same-context module dependency | `uses SiblingModule` or qualified module name |
| Cross-context contract dependency | `uses OtherContext.Interface` plus `Consumer -> OtherContext via Interface` |
| Source file owned by a module | `files [:file_atom]` |
| Existing physical path differs from convention | `.bom` explicit mapping |
| Root executable outside module tree | architecture-level `entrypoint :name` |
| External library or standard library import | no Bound relation; keep it in implementation details |

When `bound compile` reports an undeclared import, first determine whether the import crosses a real architectural boundary. If yes, add or correct the relation and `uses` declaration. If no, move ownership or refine the module boundary; do not add a blanket dependency just to silence the compiler.

## Validation and review

Run:

```sh
bound compile path/to/architecture.bo > architecture.json
bound review path/to/architecture.bo
```

Treat failures as specification feedback:

- Parse failures indicate invalid syntax or malformed imports.
- Validation failures indicate missing owners, contracts, types, mappings, or relationships.
- Analyze failures indicate that the implementation and specification disagree about imports, paths, files, or quality rules.

Use the JSON output to inspect resolved modules and mappings. Use the HTML review to ask:

- Can a reader identify who owns each capability and source file?
- Are dependency arrows intentional and directional?
- Are public contracts distinct from private implementation modules?
- Does the diagram explain the system without relying on package names alone?
- For an existing system, does it reveal violations rather than hide them?
