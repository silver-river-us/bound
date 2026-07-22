# Bound

Bound is a language-neutral architecture contract language.

It declares contexts, implementation targets, exposed contracts, and allowed relationships. The first implementation validates the model and renders a Structurizr DSL workspace. An implementation target has a language and a locator, so the same architecture can be realized in Go, Rust, Python, TypeScript, or another backend.

```bo
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
