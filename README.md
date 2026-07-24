# Bound

Bound is a language-neutral architecture contract language.

It declares contexts, one architecture-level implementation target, exposed
contracts, and allowed relationships. The first implementation validates the
model and renders a Structurizr DSL workspace. An implementation target has a
language and a root locator, so another architecture can be realized in Go,
Rust, Python, TypeScript, or another backend without leaking implementation
paths into its domain model.

```bo
"""
The architecture description is attached to the architecture AST node.
"""
architecture Commerce do
  implementation go "./"

  context Orders do
    interface OrderPort do
      behavior Place(orderID string, amount int) Order
    end
    exposes OrderPort
  end

  context Customers do
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

The architecture declaration stays in a `.bo` file. Source ownership is kept in
a separate language-qualified `.go.bom` map imported by the architecture; this keeps implementation
layout separate from domain design:

```bo
# commerce.bo
architecture Commerce do
  implementation go "./"
  import "commerce/commerce.go.bom"
  # contexts, objects, interfaces, and relationships...
end
```

The `.go.bom` maps each Go source file to exactly one private module. Named
entrypoints are bound to their implementation files there too:

```go.bom
map Commerce do
  "internal/orders/order.go" -> Orders.Internal.Orders
  entrypoint Commerce "cmd/commerce/main.go" -> App.Cmd.Commerce
end
```

Imports are optional, and paths are resolved relative to the importing `.bo`.
The Go checker requires every Go source file under the architecture implementation to
appear exactly once in the imported map. Functions and ordinary language
declarations may coexist in a mapped file; the architectural unit is the
private module, not an individual Go function.

Behaviors have explicit method names and may reference entities and values in
their language-neutral signatures:

```bo
interface ActivitySource do
  behavior activity(organization, timeWindow) returns Activity[]
end
```

The architecture declares its implementation language and source root once.
Contexts and interfaces remain language-neutral, while the `.bom` map provides
the precise ownership of each source file:

```bo
architecture GitHubDaily do
  implementation go "./"
  import "github-daily.go.bom"

  context DailyReporting do
    interface GithubActivity do
      behavior organizations() returns Organization[]
    end
    exposes GithubActivity

    module DailyReporting do
      module Activity do
        implements GithubActivity

        module Github do
          uses GithubActivity
        end
      end

      module DailyReport do
        uses GithubActivity
        entrypoint GithubDaily
      end
    end
  end
end
```

Module names conventionally produce snake-case folders, so the example declares
`daily_reporting/activity/github` without embedding paths in the DSL.
`implements` binds private code to a public contract, while `uses` permits a
dependency on an interface or another private module. Nested source folders
must have matching nested module declarations. The Go backend conventionally
resolves `entrypoint GithubDaily` beneath `cmd/github-daily`.

## Try it

```sh
go run ./cmd/bound check --root examples/commerce examples/commerce.bo
go run ./cmd/bound render examples/commerce.bo
(cd examples/commerce && go test ./... && go run ./cmd/commerce)
go test ./...
```

The architecture model is intentionally independent of implementation
languages. Backends translate module and entrypoint names into their native
folder and executable conventions.

The Go backend resolves the single implementation locator relative to the
analyzed source root. Standard-library and third-party imports are ignored;
cross-context imports must have a declared relationship in the `.bo` model.

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
    ├── activity/
    │   ├── *.go      # organizations, events, activities, search results
    │   └── github/   # GitHub API client and activity sources
    └── daily_report/
    ├── report.go     # report behavior
    └── cmd/           # explicit entry point
```

The `.go.bom` maps each file to its qualified private module. The checker
enforces the conventional module folders and rejects imports that lack a
matching `uses` declaration.
