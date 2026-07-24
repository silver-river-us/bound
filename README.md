# Bound

Bound is a language-neutral architecture contract language.

It declares contexts, implementation targets, exposed contracts, and allowed relationships. The first implementation validates the model and renders a Structurizr DSL workspace. An implementation target has a language and a locator, so the same architecture can be realized in Go, Rust, Python, TypeScript, or another backend.

```bo
"""
The architecture description is attached to the architecture AST node.
"""
architecture Commerce do
  context Orders do
    implementation go "./internal/orders"
    interface OrderPort do
      behavior Place(orderID string, amount int) Order
    end
    exposes OrderPort
  end

  context Customers do
    implementation rust "./crates/customers"
    interface CustomerPort do
      behavior Find(customerID string) Customer
    end
    exposes CustomerPort
  end

  Orders -> Customers via CustomerPort
end
```

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
- `behavior` declares what an interface or module can do. It is intentionally
  neutral between commands, queries, and language-specific functions.

Use `entity` when identity, ownership, lifecycle, or invariants matter. Use
`value` for concepts such as time windows, addresses, measurements, and other
descriptive data. The distinction affects equality, mutation, persistence, and
aggregate boundaries; it is not a claim about the eventual implementation
type.

```bo
entity Organization do
  state :login :string
end

value TimeWindow do
  state :since :timestamp
  state :until :timestamp
end
```

An entity or value can expose behaviors through an interface:

```bo
interface ActivitySource do
  behavior activity(organization, timeWindow) returns Activity[]
end
```

The architecture declaration stays in a `.bo` file. Source ownership is kept in
a separate language-qualified `.go.bom` map imported by the architecture; this keeps implementation
layout separate from domain design:

```bo
# commerce.bo
architecture Commerce do
  import "commerce/commerce.go.bom"
  # contexts, objects, interfaces, and relationships...
end
```

The `.go.bom` maps each Go source file to exactly one context or exposed module.
Entry points are explicit mappings too:

```go.bom
map Commerce do
  "internal/orders/order.go" -> Orders
  entrypoint "cmd/commerce/main.go" -> App
end
```

Imports are optional, and paths are resolved relative to the importing `.bo`.
The Go checker requires every Go source file under a declared implementation to
appear exactly once in the imported map. A mapping can target a context or one
of its exposed interfaces, which lets architecturally significant modules such
as `GithubActivity` and `DailyReport` own source files. Functions and ordinary
language declarations may coexist in a mapped file; the architectural unit is
the mapping target, not an individual Go function.

Behaviors have explicit method names and may reference entities and values in
their language-neutral signatures:

```bo
interface ActivitySource do
  behavior activity(organization, timeWindow) returns Activity[]
end
```

The architecture can declare every module namespace and its implementation
folder directly in `.bo`. The Go checker requires every mapped file to be
physically inside the declared module folder:

```bo
context DailyReporting do
  implementation go "./"
  interface GithubActivity do
    behavior activity(organization, timeWindow) returns Activity[]
  end
  exposes GithubActivity

  interface DailyReport do
    behavior render(organizations, activities, warnings)
  end
  exposes DailyReport
end

module GithubActivity do
  implementation go "./daily_reporting/github_activity"
end

module GithubActivity.GitHub do
  implementation go "./daily_reporting/github_activity/github"
end

module DailyReport do
  implementation go "./daily_reporting/daily_report"
end

module DailyReport.Command do
  implementation go "./daily_reporting/daily_report/cmd/github-daily"
end
```

This makes the folder structure part of the architecture contract. A source
file cannot silently belong to a different module just because it imports the
right package.

## Try it

```sh
go run ./cmd/bound check --root examples/commerce examples/commerce.bo
go run ./cmd/bound render examples/commerce.bo
(cd examples/commerce && go test ./... && go run ./cmd/commerce)
go test ./...
```

The architecture model is intentionally independent of implementation languages. Future backends will inspect the corresponding source tree and enforce that its imports or module dependencies obey the declared relationships.

The Go backend currently resolves implementation locators relative to the analyzed source root. Standard-library and third-party imports are ignored; cross-context imports must have a declared relationship in the `.bo` model.

Interfaces are architecture contracts, not tied to one implementation language. A Go backend can realize them as ordinary Go interfaces, as shown in `examples/commerce/internal/orders/order.go` and `examples/commerce/internal/customers/customer.go`.

## GitHub daily activity example

The `examples/github-daily` program is declared by [`github-daily.bo`](examples/github-daily/github-daily.bo). It discovers all organizations visible to the authenticated GitHub user, reads each organization's activity sources for the last 24 hours, and writes a Markdown report. The program validates that `.bo` architecture before it runs.

```sh
go run ./examples/github-daily/daily_reporting/daily_report/cmd/github-daily
go run ./examples/github-daily/daily_reporting/daily_report/cmd/github-daily -since 48h -output -
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
└── daily_reporting/
    ├── github_activity/
    │   ├── *.go      # organizations, events, activities, search results
    │   └── github/   # GitHub API client and activity sources
    └── daily_report/
    ├── report.go     # report behavior
    └── cmd/           # explicit entry point
```

The `.go.bom` maps both namespaces to the `GithubActivity` and `DailyReport`
modules, so a file cannot drift into an unrelated architectural folder.
