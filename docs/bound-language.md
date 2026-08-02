# Bound language reference

This document is the canonical reference for the currently supported `.bo`
architecture language. Bound is line-oriented: whitespace around a statement is
ignored, statements do not use semicolons, and every block is closed with
`end`.

## Lexical rules

- Identifiers begin with `A-Z`, `a-z`, or `_`, followed by letters, digits, or
  `_`. Qualified names join identifiers with `.`.
- Types use an identifier or qualified name, optionally followed by `[]` for a
  list. Built-in contract types are `any`, `bool`, `decimal`, `float`, `int`,
  `string`, and `timestamp`.
- A `#` starts a comment and removes the remainder of that line.
- A documentation block is a pair of lines containing `"""`. Its contents are
  attached to the next architecture, context, interface, domain type,
  behavior, or relationship declaration.
- File paths and implementation locators are quoted strings. Import paths must
  be relative and must remain below the directory containing the importing
  file.

## Architecture grammar

The following is an intentionally compact EBNF summary. The semantic rules
below define constraints that cannot be expressed by the grammar alone.

```ebnf
architecture = "architecture" identifier "do",
               implementation,
               { architecture-item },
               "end" ;

architecture-item = import | entrypoint | quality | domain-type |
                    context | relationship ;

implementation = "implementation" identifier quoted-string ;
import = "import" identifier "from" quoted-string ;
entrypoint = "entrypoint" ":" identifier ;
relationship = identifier "->" identifier [ "via" identifier ] ;

context = "context" identifier "do",
          { context-item },
          "end" ;
context-item = import | "exposes" identifier | interface | module ;

interface = "interface" identifier "do",
            { interface-item },
            "end" ;
interface-item = domain-type | behavior ;

domain-type = ( "entity" | "value" ) identifier "do",
              { state | behavior },
              "end" ;
state = "state" ":" identifier ":" type ;
behavior = "behavior" identifier "(" [ parameters ] ")",
           [ "returns" type ] ;
parameters = parameter { "," parameter } ;
parameter = identifier type ;

a-module = "module" identifier "do",
           { module-item | a-module },
           "end" ;
module-item = "implements" qualified-name |
              "uses" qualified-name |
              "file" ":" identifier |
              "files" "[" [ file-atom { "," file-atom } ] "]" |
              "entrypoint" identifier [ quoted-string ] ;
file-atom = ":" identifier ;

quality = "quality" "do",
          { quality-limit | quality-rules },
          "end" ;
quality-limit = ( "max_function_lines" |
                  "max_cyclomatic_complexity" |
                  "max_nesting_depth" |
                  "max_parameters" |
                  "max_file_lines" ), integer ;
quality-rules = "rules" "do",
                [ "one_declaration_kind_per_file" ],
                "end" ;
```

The parser accepts `module` blocks only inside a context. The top-level
`entrypoint :name` form creates a root executable mapping; module entrypoints
may additionally provide an explicit source path.

## Imports

An import is resolved relative to the importing file:

```bo
import ReportingApplication from "contracts/reporting_application.bo"
import ExistingSources from "legacy/app.bom"
```

- `.bo` imports select a named architecture or contract fragment.
- `.bom` imports select a named source map for an existing repository.
- Import names must match the architecture, interface, or map declared by the
  imported file.
- Import cycles are rejected.
- A `.bo` fragment contributes the selected contract; contexts, modules,
  relationships, implementation metadata, and source mappings remain in the
  importing architecture.

## Semantic rules

- An architecture has exactly one implementation language and source locator.
  The language must have a compiler backend when implementation analysis runs.
  The current implementation backends are Go, Ruby, and Python.
- Names must be unique within their scope. This includes contexts, interfaces,
  domain types, behaviors, modules, states, and behavior parameters.
- `entity` represents an identity-bearing domain object and may declare
  behaviors. `value` represents descriptive data and cannot declare behaviors.
- Architecture-level domain types may be referenced by architecture behaviors.
  Contract types may reference primitives, types in the same interface, or
  exposed types in a related context using qualified names.
- A context can expose only interfaces declared in that context.
- `implements` must name an interface in the containing context. `uses` may name
  a local module, a parent/child module, a same-context module, a local
  interface, or an exposed interface in a related context.
- A relationship connects two different contexts. `via` must identify an
  interface exposed by the destination context, and that interface must be
  visible through the declared relationship.
- Module names are converted to conventional folder names. Sibling modules
  cannot collapse to the same folder name, and nested modules produce nested
  source paths.
- Every declared module file maps to one generated implementation source file.
  Paths are relative, normalized, and cannot escape the implementation root.
- Every module entrypoint must resolve to exactly one source mapping. The root
  entrypoint is derived from the implementation language unless explicitly
  mapped.
- Quality limits must be positive integers. Quality checks are disabled when
  no limits or rules are declared.

## Source maps (`.bom`)

A source map describes an existing repository whose physical paths do not match
Bound's conventional module folders:

```bom
map ExistingSources do
  "app/services/reporting.rb" -> Lib.Reporting
  entrypoint main "bin/report" -> Boundary.CLI
end
```

The map name must match the symbol imported by the `.bo` file. Mapped paths are
relative to the implementation root, and each path may be mapped only once.

## Compilation model

`bound compile` parses imports, validates the language-neutral model, resolves
the implementation root, and runs the selected backend. Go, Ruby, and Python
validate mapped source ownership and file existence; Go additionally checks
package imports, while Ruby and Python do not infer dynamic-language dependencies. `bound review` runs the
same checks before generating its HTML review. The review includes section
navigation, a compact count summary, and a domain-model scope selector for
architecture-level types and types owned by each context's interfaces. A parser
or model error is not a backend-specific warning: compilation fails with a
phase-aware diagnostic.
