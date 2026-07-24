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
      operation Place(orderID string, amount int) Order
    end
    exposes OrderPort
  end

  context Customers do
    implementation rust "./crates/customers"
    interface CustomerPort do
      operation Find(customerID string) Customer
    end
    exposes CustomerPort
  end

  Orders -> Customers via CustomerPort
end
```

Triple-quoted blocks are documentation nodes. A block immediately before an
architecture, context, interface, operation, or relationship is stored on that
AST node and can be rendered by future documentation backends.

Domain objects and their attributes are declared explicitly:

```bo
object Activity do
  attribute :repository :string
  attribute :occurred_at :timestamp
end
```

Attributes use Ruby-like symbols for both the field name and its language-neutral
type: `attribute :name :type`.

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

The `.go.bom` maps each Go source file to exactly one context. Entry points are
explicit mappings too:

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

Operations have explicit method names and may reference those objects in their
language-neutral signatures:

```bo
interface ActivitySource do
  operation activity(organization, timeWindow) returns Activity[]
end
```

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
go run ./examples/github-daily/daily_report/cmd/github-daily
go run ./examples/github-daily/daily_report/cmd/github-daily -since 48h -output -
```

Authentication uses `GITHUB_TOKEN` when set, otherwise the token from `gh auth token`. Reports are written to `reports/github-activity-YYYY-MM-DD.md` by default.

The report combines three GitHub sources: organization events, commit search,
and issue/pull-request search. The latter represents an updated issue or pull
request, not every individual comment or review actor. GitHub audit-log access
is organization-admin scoped, so the program reports everything available to
the authenticated token rather than claiming universal private-audit coverage.
