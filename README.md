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

The complete syntax and semantic rules are documented in the
[Bound language reference](docs/bound-language.md). Bound also includes a
stdio language server for live `.bo` diagnostics and completion in Zed; see the
[Zed setup guide](docs/zed.md). The planned vocabulary and
validation rules for asynchronous systems, queues, databases, events,
polymorphism, and external systems are documented in
[Architecture concepts](docs/architecture-concepts.md).

`bound compile` is the compiler entry point. It parses the architecture, resolves
relative imports, validates the architecture model, derives the implementation
source root, and runs the backend checks. For Go, those checks include source
ownership, conventional module paths, quality rules, and actual package imports
against declared `uses` and context relationships. Ruby and Python validate mapped
source ownership, file existence, and conventional module paths without inferring
dynamic-language dependency graphs.

```sh
bound compile examples/github-activity/github-activity.bo > architecture.json
bound review examples/github-activity/github-activity.bo
```

`compile` prints the resolved compiler IR as JSON. `review` writes and opens a
standalone review page containing the context, component, interaction, contract/module,
and source-ownership diagrams. The page uses Mermaid in the browser and
includes zoom and fullscreen controls. The render package also supports Markdown,
SVG, and HTML exports; HTML callers can point Mermaid at a local ESM asset or
embed a trusted UMD asset, and can opt into print-friendly CSS. The SVG export
is self-contained and selectable, preserving Mermaid source in metadata for
offline archival; browser-rendered SVG remains available through the HTML
review. The default `MermaidHTML` behavior remains CDN-backed for compatibility.

The compiler reports failures by phase (`parse`, `validate`, or `analyze`) so a
future backend can add language-specific checks without coupling them to the
Bound parser. The architecture model is the compiler's intermediate
representation and is shared by validation, dependency analysis, and rendering
backends.

Bound also exposes the compiler as a Go library for tools that do not want to
shell out to the CLI:

```go
program, err := bound.Compile("architecture.bo", bound.Options{})
if err != nil {
    // Inspect *bound.Error for phase-aware diagnostics.
}
ir, err := program.JSON()
```

The public package keeps the compiler implementation private. Its
`SchemaVersion` constant identifies the version emitted by `Program.JSON`.
Within a schema version, existing IR fields retain their meaning and new fields
may be added. Consumers should ignore unknown JSON fields; breaking changes
increment the schema version.

## Automated architecture refresh

The `Refresh architecture with agent` workflow runs weekly or on demand. It
asks an agent to inspect source-backed `.bo`/`.bom` files, run the repository
checks, and open a pull request only when it finds evidence of drift. The agent
instructions live in [`.github/architecture-agent.md`](.github/architecture-agent.md).
Set the repository's `OPENAI_API_KEY` secret to enable the scheduled job. Every
change remains a normal pull request for human review; the workflow does not
merge or push directly to the default branch. Use the
[architecture PR review checklist](docs/architecture-review.md) when reviewing
those changes.

## Installation and builds

Bound requires Go 1.22 or newer. The compiler and library are developed and
validated on the platforms supported by that Go release; the published CLI
artifacts currently target Linux (`amd64`, `arm64`), macOS (`amd64`, `arm64`),
and Windows (`amd64`). Use a release artifact when you want a standalone
executable, or install/build from source when working from a checkout.

Run the CLI from a repository checkout with the root launcher:

```sh
./bound compile examples/github-activity/github-activity.bo
```

The launcher builds `src` into `target/go/bound`. For a manually placed
standalone binary, build the command package directly:

```sh
go build -trimpath -o target/go/bound ./src
./target/go/bound compile examples/github-activity/github-activity.bo > architecture.json
```

Release archives are named `bound-<version>-<os>-<arch>`, where `<version>` is
the exact `vMAJOR.MINOR.PATCH` tag without the leading `v` in the filename. The
release also contains a SHA-256 checksum file and a
`bound-architecture-specs-<version>.tar.gz` archive containing the
source-controlled skill and its agent configuration. See
[`docs/releasing.md`](docs/releasing.md) for the versioning, changelog, and
compiler IR compatibility policy.

## Try it

The repository keeps production Go code under `src/`: the CLI entrypoint is
`src/main.go`, and the compiler packages live alongside it. Tests are kept in
the sibling `tests/` tree so production packages contain no test files.

Use the repository launcher, following the same pattern as qfile:

```sh
./bound compile examples/github-activity/github-activity.bo > architecture.json
./bound review examples/github-activity/github-activity.bo
```

The launcher builds a trimmed binary at `target/go/bound` and reuses that
location on subsequent invocations. It honors the active ASDF Go installation
when ASDF is available. `go run ./src` remains available as a low-level
fallback when the launcher is not suitable.

Run the test suite with:

```sh
go test ./...
```

The architecture model is intentionally independent of implementation
languages. Backends translate module and entrypoint names into their native
folder and executable conventions.

The Go backend resolves the implementation locator relative to the analyzed
source root. It discovers the root module, nested `go.mod` files, and modules
listed by a `go.work` file, then checks each module's package imports. Standard-
library and third-party imports are ignored; cross-context imports must have a
declared relationship in the `.bo` model. The source root may be outside the
Bound repository, but mapped paths remain relative to that root.

For Ruby and Python, Bound deliberately stops at explicit source ownership and the
architecture's declared relationships. It does not infer a dependency graph
from `require`, Rails autoloading, or Bundler metadata: those mechanisms are
dynamic and would produce incomplete or environment-dependent results, while
current Ruby and Python architectures have not required implementation-level
dependency checking beyond ownership. If a real Ruby architecture needs a static
dependency backend, it should be added as a separate, evidence-driven feature
rather than silently changing the meaning of existing ownership checks.

## Greenfield design example

[`examples/greenfield/checkout.bo`](examples/greenfield/checkout.bo) shows a
design-first architecture with contexts, contracts, domain types, and
relationships before implementation code exists. Validate this mode through the
library API with `bound.Options{SkipImplementation: true}`; implementation
checks can be enabled once the source tree is created.

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

## Stitch Bank architecture example

The [`examples/stitch-bank`](examples/stitch-bank/stitch-bank.bo) specification
reverse-engineers the sibling `../stitch_bank` repository. It models the
applicant and operations experiences, Rails origination domain, provider
integrations, documents/compliance, and tenant platform. Deployment
infrastructure is intentionally outside this application architecture.

```sh
bound compile examples/stitch-bank/stitch-bank.bo > stitch-bank.json
bound review examples/stitch-bank/stitch-bank.bo
```

The Ruby backend validates the explicit Rails source ownership map. Rails
autoloading does not expose a reliable static import graph, so cross-context
dependencies are represented by the contracts and relations in the `.bo`
model.

## qfile architecture example

The [`examples/qfile`](examples/qfile/qfile.bo) specification reverse-engineers
the sibling `../qfile` Go CLI. It models the Boundary, Domain, Infrastructure,
and Command contexts, including search, conversation history, knowledge,
memory, filesystem, Codex process, and source-ownership modules.

```sh
bound compile examples/qfile/qfile.bo > qfile.json
bound review examples/qfile/qfile.bo
```

The generated review includes the qfile domain objects (`SearchEntry`,
`SearchMatch`, `Conversation`, `KnowledgeDocument`, and `Manifest`) alongside
the module and dependency diagrams.
