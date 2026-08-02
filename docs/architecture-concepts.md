# Architecture concepts beyond modules

This document defines the vocabulary for architecture elements that are not
ordinary synchronous modules. The current compiler models contexts, contracts,
modules, relationships, and source ownership. These rules provide the stable
meaning for future syntax and IR fields without treating implementation details
as domain concepts.

## Element kinds

- **External system**: a system outside the implementation root that the
  architecture calls or receives data from. It has an owner/context, a named
  contract, and an explicit direction.
- **Database**: durable state owned by one context. Other contexts access it
  through an exposed contract, never through a direct shared-storage edge.
- **Queue**: an asynchronous transport with a named message contract. Producers
  publish messages; consumers subscribe to them. A queue is not a synchronous
  module dependency.
- **Event**: an immutable message describing something that happened. Events
  have a producer, a message type, and zero or more subscribers.
- **Polymorphic contract**: a contract whose operation accepts or returns one
  of a declared set of compatible variants. The variants must remain explicit
  in the architecture model.

## Relationship rules

Synchronous relationships use the existing `A -> B via Contract` form. Future
asynchronous relationships must identify their transport and message type,
for example conceptually `A -[publishes Event via Queue]-> B`; they must not be
encoded as ordinary `uses` dependencies.

The compiler should reject:

- a queue without exactly one declared message contract;
- a consumer that subscribes to a message it does not expose;
- a database with more than one owning context;
- direct module dependencies that bypass a context's exposed contract;
- polymorphic variants that are undeclared or duplicate one another;
- synchronous cycles unless explicitly allowed by a future policy option.

## IR expectations

When these concepts become syntax, each element should have a stable kind,
qualified name, owner, description, and contract reference. Relationships should
carry their interaction mode (`sync`, `event`, or `queue`), direction, and
contract/transport metadata. Existing consumers must continue to treat unknown
kinds and fields as additive IR data until a schema version introduces a
breaking semantic change.

Until parser support is added, external systems, databases, queues, and events
should be represented as documented contexts or interfaces rather than by
inventing ad-hoc module names.
