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
  import contracts from "contracts/github_activity.bo"
  import contracts from "contracts/daily_report.bo"

  exposes GitHubActivity
  exposes DailyReport
end
```

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
architecture so its structural and data-flow view stays visible.

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
architecture GitHubDaily do
  implementation go "./"

  context Reporting do
    interface GitHubActivity do
      behavior organizations() returns Organization[]
    end
    exposes GitHubActivity

    module Reporting do
      module Activity do
        implements GitHubActivity

        module GitHub do
          uses GitHubActivity
          files [:client]
        end
      end

      module DailyReport do
        uses GitHubActivity
        entrypoint GitHubDaily
      end
    end
  end
end
```

Module names conventionally produce snake-case folders, so the example declares
`reporting/activity/github` without embedding paths in the DSL.
`implements` binds private code to a public contract, while `uses` permits a
dependency on an interface or another private module. Nested source folders
must have matching nested module declarations. The Go backend conventionally
resolves `entrypoint GitHubDaily` beneath `command/main.go`.

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

The `examples/github-daily` program is declared by [`github-daily.bo`](examples/github-daily/github-daily.bo). It discovers all organizations visible to the authenticated GitHub user, reads each organization's activity sources for the last 24 hours, and writes a Markdown report. The program validates that `.bo` architecture before it runs.

```sh
go run ./examples/github-daily/reporting/daily_report/command
go run ./examples/github-daily/reporting/daily_report/command -since 48h -output -
```

Authentication uses `GITHUB_TOKEN` when set, otherwise the token from `gh auth token`. Reports are written to `reports/github-activity-YYYY-MM-DD.md` by default.

The report combines three GitHub sources: organization events, commit search,
and issue/pull-request search. The latter represents an updated issue or pull
request, not every individual comment or review actor. GitHub audit-log access
is organization-admin scoped, so the program reports everything available to
the authenticated token rather than claiming universal private-audit coverage.

The implementation namespace follows the architecture:

```text
github-daily/
└── reporting/
    ├── activity/
    │   ├── *.go      # organizations, events, activities, search results
    │   └── github/   # GitHub API client and activity sources
    └── daily_report/
        ├── report.go # report behavior
        └── command/  # conventional named entrypoint
```

The checker requires every Go source file to have exactly one module
declaration, enforces the conventional module folders, and rejects imports
that lack a matching `uses` declaration. Legacy `.bom` maps remain supported
for imported architectures.
