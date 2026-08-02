# Automated architecture PR review

Architecture refreshes are intentionally proposed as pull requests, not merged
by automation. A reviewer should check every generated PR before merging:

- Confirm each `.bo`/`.bom` change is supported by the checked-out source.
- Confirm source revisions and inspection dates are present for reverse-engineered
  examples.
- Run `go test ./...`, `go vet ./...`, and compile every available example.
- Review ownership, context boundaries, exposed contracts, and dependency
  direction separately from formatting changes.
- Reject redesigns, speculative domains, or changes that only make validation
  pass without matching the source.
- Record uncertainty in the PR discussion and request a follow-up architecture
  task when the source is ambiguous.

The scheduled workflow remains safe only when its generated pull requests are
reviewed using this checklist. Repository branch protection should require at
least one human approval for `.bo`, `.bom`, and architecture workflow changes.
