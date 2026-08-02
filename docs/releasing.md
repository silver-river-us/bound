# Releasing Bound

This document defines the release conventions for the CLI, library, compiler IR,
and the `bound-architecture-specs` skill.

## Versioning

- Release tags use `vMAJOR.MINOR.PATCH` and follow Semantic Versioning.
- The tag is the release source of truth. The workflow removes the leading `v`
  only when naming archive files.
- A major version is for incompatible public API or compiler IR changes. A
  minor version is for backwards-compatible functionality. A patch version is
  for backwards-compatible fixes and documentation/maintenance changes.
- Do not create a release tag until `CHANGELOG.md` has an entry for that
  version. Move the relevant `Unreleased` entries into the new version section,
  add the release date, and start a new empty `Unreleased` section.
- Release tags run `.github/workflows/release.yml`. The workflow publishes
  cross-platform CLI archives, `SHA256SUMS`, and the skill archive.

The current workflow publishes these targets:

| OS | Architectures |
| --- | --- |
| Linux | `amd64`, `arm64` |
| macOS | `amd64`, `arm64` |
| Windows | `amd64` |

These are release targets, not a statement that other Go-supported platforms
cannot build from source. Source builds require Go 1.22 or newer; see the
installation section in the README.

## Changelog conventions

Use the Keep a Changelog headings, as applicable:

- **Added** for new capabilities.
- **Changed** for behavior or compatibility changes.
- **Deprecated** for supported features planned for removal.
- **Removed** for removed features.
- **Fixed** for corrections.
- **Security** for security fixes.

Describe user-visible behavior and migration requirements. Keep implementation
refactors out unless they affect users, release artifacts, or maintenance
support. Each release entry should mention compiler IR changes explicitly, even
when the answer is “none.”

## Compiler IR compatibility

`Program.JSON` emits a `schema_version`; the current schema is version `1`.
Within a schema version:

- Existing fields retain their meaning and type.
- New fields may be added.
- Consumers must ignore unknown fields.

A change is breaking when it removes a field, changes the meaning or type of an
existing field, changes required semantics, or makes previously valid JSON
unreadable. Breaking changes increment `schema_version` and must be called out
in the changelog with migration notes. The compiler/library release version and
the IR schema version are independent: a normal Bound release does not imply an
IR migration.

When incrementing the IR schema, document the old and new versions, the affected
fields, and the consumer action required to migrate. Update the CI assertion and
any fixtures or documentation that intentionally pin the schema.

## Skill package contents

The release workflow creates
`bound-architecture-specs-<version>.tar.gz` from
`integrations/skills/bound-architecture-specs`. It includes:

- `SKILL.md`;
- `references/spec-writing.md`; and
- `agents/openai.yaml`.

Keep the skill self-contained and update its frontmatter, reference links, and
agent configuration together. The package is an additive release asset; it is
not copied into the Go module and does not change the CLI build.

## Release checklist

1. Update `CHANGELOG.md` and `docs/releasing.md` if the policy changed.
2. Run the checks documented in the README (`go test ./...`, `go vet ./...`,
   and the example compilation check).
3. Review the generated diff and confirm no Go source changes are part of a
   documentation-only or maintenance release.
4. Create and push a `vMAJOR.MINOR.PATCH` tag.
5. Verify the GitHub release contains all expected archives, `SHA256SUMS`, and
   the skill package.
6. After publishing, replace the release entry's `Unreleased` link or notes as
   appropriate for the hosting repository.
