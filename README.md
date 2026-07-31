# Bound

Bound is a language-neutral architecture contract language.

It declares contexts, one architecture-level implementation target, exposed
contracts, and allowed relationships. The first implementation validates the
model and renders a Structurizr DSL workspace. An implementation target has a
language and a root locator, so another architecture can be realized in Go,
Rust, Python, TypeScript, or another backend without leaking implementation
paths into its domain model.

Triple-quoted blocks are documentation nodes. A block immediately before an
architecture, context, interface, behavior, or relationship is stored on that
AST node and can be rendered by future documentation backends.

## DDD vocabulary

Bound uses DDD-oriented terms for domain design:

- `entity` represents something with identity and a lifecycle. Two entities
  with equal state may still be different entities.
- `value` represents something defined entirely by its contents. Equal values
  are interchangeable and are generally immutable or replaced as a whole.
- `state` declares the data that makes up an entity or value. In DDD, state is
  the collection of attributes; `state` is the architectural keyword in Bound.
- `behavior` declares what an entity or interface can do. Entities may declare
  behavior alongside state; values are state-only and behavior-free.

Use `entity` when identity, ownership, lifecycle, or invariants matter. Use
`value` for concepts such as time windows, addresses, measurements, and other
descriptive data. The distinction affects equality, mutation, persistence, and
aggregate boundaries; it is not a claim about the eventual implementation
type.

```bo
interface ActivitySource do
  entity Organization do
    state :login :string
  end

  value TimeWindow do
    state :since :timestamp
    state :until :timestamp
  end

  behavior activity(organization Organization, window TimeWindow) returns Activity[]
end
```

Types are part of the interface that exposes them. Other interfaces reference
them with qualified names such as `ActivitySource.Organization`.

Large contracts can live in standalone `.bo` fragments and be imported by the
context that owns them:

```bo
# reporting.bo
context Reporting do
  import GitHubActivity from "contracts/github_activity.bo"
  import DailyReport from "contracts/daily_report.bo"

  exposes GitHubActivity
  exposes DailyReport
end
```

The name before `from` is the exact symbol being imported. A contract fragment
may define several interfaces, but each import selects one interface by name.
Architecture-level `.bo` and `.bom` imports use the same rule: the imported
name must match the architecture or map declared by the file.

```bo
# contracts/github_activity.bo
interface GitHubActivity do
  value TimeWindow do
    state :since :timestamp
    state :until :timestamp
  end

  behavior activities(organization Organization, window TimeWindow) returns ActivityFeed
end
```

Interface fragments may contain documentation, interfaces, their entities and
values, behaviors, and other relative fragment imports. Contexts, modules,
relationships, implementation metadata, and source maps remain in the main
architecture so its structural and data-flow view stays visible. Fragment
imports are also named and resolve only the requested interface.

Behaviors have explicit method names and may reference entities and values in
their language-neutral signatures:

```bo
interface ActivitySource do
  behavior activity(organization Organization, window TimeWindow) returns Activity[]
end
```

The architecture declares its implementation language and source root once.
Contexts and interfaces remain language-neutral, while module `file`
declarations provide the precise ownership of each source file:

```bo
architecture GitHubActivity do
  implementation go "./"
  entrypoint :main

  context Boundary do
    module Boundary do
      module CLI do
        uses Lib.ReportingApplication
      end
    end
  end

  context Infrastructure do
    module Infrastructure do
      module GitHub do
        implements GitHubActivity
      end
    end
  end

  context Lib do
    module Lib do
      module Activity do
        implements GitHubActivity
      end
      module Reporting do
        implements ReportingApplication
        uses GitHubActivity
        uses DailyReport
      end
    end
  end
end
```

Module names conventionally produce snake-case folders, so the example declares
`boundary/cli`, `infrastructure/github`, `lib/activity`, and
`lib/reporting` without embedding paths in the DSL.
`implements` binds private code to a public contract, while `uses` permits a
dependency on an interface or another private module. Nested source folders
must have matching nested module declarations. The architecture-level
`entrypoint :main` derives `main.go` from the target language and keeps the
executable outside the component module tree.

The example uses three explicit contexts as a screaming architecture:
`Boundary` contains replaceable user interfaces, `Infrastructure` contains
external-system adapters and parsing, and `Lib` contains the `Activity` and
`Reporting` bounded contexts plus their use-case orchestration.
The lib returns structured reporting data; Markdown rendering, stdout/file
output, and operational logging remain in the CLI boundary.

Architecture files can also enforce implementation quality limits. Limits are
disabled when omitted; each declared limit must be a positive integer:

```bo
quality do
  max_function_lines 15
  max_cyclomatic_complexity 10
  max_nesting_depth 4
  max_parameters 5
  max_file_lines 200
  rules do
    one_declaration_kind_per_file
  end
end
```

The Go analyzer reports the source file and function when a mapped file exceeds
its file-length limit or a function exceeds its line, cyclomatic-complexity,
nesting-depth, or parameter limit. The declaration rule keeps top-level types,
functions, constants, and variables from being mixed in one source file.

## Compiler

`bound compile` is the compiler entry point. It parses the architecture, resolves
relative imports, validates the architecture model, derives the implementation
source root, and runs the backend checks. For Go, those checks include source
ownership, conventional module paths, quality rules, and actual package imports
against declared `uses` and context relationships.

```sh
bound compile examples/github-activity/github-activity.bo > architecture.json
bound review examples/github-activity/github-activity.bo
```

`compile` prints the resolved compiler IR as JSON. `review` writes and opens a
standalone review page containing the context, component, interaction, contract/module,
and source-ownership diagrams. The page uses Mermaid in the browser and
includes zoom and fullscreen controls; opening it requires access to the
Mermaid CDN.

The compiler reports failures by phase (`parse`, `validate`, or `analyze`) so a
future backend can add language-specific checks without coupling them to the
Bound parser. The architecture model is the compiler's intermediate
representation and is shared by validation, dependency analysis, and rendering
backends.

## Try it

```sh
go test ./...
```

The architecture model is intentionally independent of implementation
languages. Backends translate module and entrypoint names into their native
folder and executable conventions.

The Go backend resolves the single implementation locator relative to the
analyzed source root. Standard-library and third-party imports are ignored;
cross-context imports must have a declared relationship in the `.bo` model.

## GitHub daily activity example

The `examples/github-activity` program is declared by [`github-activity.bo`](examples/github-activity/github-activity.bo). It discovers all organizations visible to the authenticated GitHub user, reads each organization's activity sources for a configurable period, and writes a Markdown report. The program validates that `.bo` architecture before it runs.

```sh
go run ./examples/github-activity
go run ./examples/github-activity -period 48h -output -
```

Authentication uses `GITHUB_TOKEN` when set, otherwise the token from `gh auth token`. Reports are written to `examples/github-activity/reports/github-activity-YYYY-MM-DD.md` by default.

The report combines three GitHub sources: organization events, commit search,
and issue/pull-request search. The latter represents an updated issue or pull
request, not every individual comment or review actor. GitHub audit-log access
is organization-admin scoped, so the program reports everything available to
the authenticated token rather than claiming universal private-audit coverage.

The executable lives at the example root, while the implementation packages
follow the architecture:

```text
bound/
└── examples/github-activity/
    ├── main.go
    ├── boundary/cli/                 # CLI adapter
    ├── infrastructure/github/        # HTTP client, JSON parsing, API adapter
    └── lib/
        ├── activity/                 # activity bounded context
        └── reporting/                # report context and use-case orchestration
```

The checker requires every Go source file to have exactly one module
declaration, enforces the conventional module folders, and rejects imports
that lack a matching `uses` declaration. Legacy `.bom` maps remain supported
for imported architectures.
