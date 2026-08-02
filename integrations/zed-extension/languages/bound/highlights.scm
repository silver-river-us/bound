; Bound declarations and control words
[
  "architecture"
  "implementation"
  "context"
  "interface"
  "entity"
  "value"
  "behavior"
  "state"
  "module"
  "implements"
  "uses"
  "exposes"
  "entrypoint"
  "import"
  "from"
  "quality"
  "rules"
  "end"
  "via"
  "returns"
  "file"
  "files"
] @keyword

; Quality limits and declaration names
[
  "max_function_lines"
  "max_cyclomatic_complexity"
  "max_nesting_depth"
  "max_parameters"
  "max_file_lines"
  "one_declaration_kind_per_file"
] @constant

; Declarations have distinct semantic roles
(architecture name: (identifier) @type)
(context name: (identifier) @namespace)
(interface name: (identifier) @type)
(domain_type name: (identifier) @type)
(module name: (identifier) @namespace)
(implementation (identifier) @type)
(behavior (identifier) @function)
(state (identifier) @property)
(parameter (identifier) @parameter)

; References and type expressions
(type_name) @type
(qualified_name) @variable
(relationship (identifier) @variable)

(integer) @number
(string) @string
(comment) @comment
(doc_block) @comment.doc

"->" @operator
":" @punctuation.delimiter
"," @punctuation.delimiter
"[" @punctuation.bracket
"]" @punctuation.bracket
"(" @punctuation.bracket
")" @punctuation.bracket
