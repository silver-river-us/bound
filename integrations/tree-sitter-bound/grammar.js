module.exports = grammar({
  name: 'bound',

  extras: $ => [/[ \t\r\n]/, $.comment],

  rules: {
    source_file: $ => repeat(choice(
      $.architecture,
      $.doc_block,
      $.comment,
    )),

    architecture: $ => seq(
      'architecture', field('name', $.identifier), 'do',
      repeat(choice(
        $.implementation,
        $.root_entrypoint,
        $.quality,
        $.import_statement,
        $.domain_type,
        $.context,
        $.relationship,
        $.doc_block,
        $.comment,
      )),
      'end',
    ),

    context: $ => seq(
      'context', field('name', $.identifier), 'do',
      repeat(choice(
        $.import_statement,
        $.exposes,
        $.interface,
        $.module,
        $.doc_block,
        $.comment,
      )),
      'end',
    ),

    interface: $ => seq(
      'interface', field('name', $.identifier), 'do',
      repeat(choice($.domain_type, $.behavior, $.doc_block, $.comment)),
      'end',
    ),

    domain_type: $ => seq(
      choice('entity', 'value'), field('name', $.identifier), 'do',
      repeat(choice($.state, $.behavior, $.doc_block, $.comment)),
      'end',
    ),

    module: $ => seq(
      'module', field('name', $.identifier), 'do',
      repeat(choice(
        $.module,
        $.module_statement,
        $.doc_block,
        $.comment,
      )),
      'end',
    ),

    quality: $ => seq(
      'quality', 'do',
      repeat(choice($.quality_rule, $.quality_rules, $.doc_block, $.comment)),
      'end',
    ),

    quality_rules: $ => seq(
      'rules', 'do',
      optional('one_declaration_kind_per_file'),
      'end',
    ),

    quality_rule: $ => seq(
      choice(
        'max_function_lines',
        'max_cyclomatic_complexity',
        'max_nesting_depth',
        'max_parameters',
        'max_file_lines',
      ),
      $.integer,
    ),

    implementation: $ => seq('implementation', $.identifier, $.string),
    root_entrypoint: $ => seq('entrypoint', ':', $.identifier),
    import_statement: $ => seq('import', $.identifier, 'from', $.string),
    exposes: $ => seq('exposes', $.identifier),
    relationship: $ => seq($.identifier, '->', $.identifier, optional(seq('via', $.identifier))),

    module_statement: $ => choice(
      seq('implements', $.qualified_name),
      seq('uses', $.qualified_name),
      seq('file', ':', $.identifier),
      seq('files', '[', optional($.file_atoms), ']'),
      seq('entrypoint', $.identifier, optional($.string)),
    ),

    file_atoms: $ => seq(':', $.identifier, repeat(seq(',', ':', $.identifier))),

    state: $ => seq('state', ':', $.identifier, ':', $.type_name),

    behavior: $ => seq(
      'behavior', $.identifier, '(', optional($.parameters), ')',
      optional(seq('returns', $.type_name)),
    ),

    parameters: $ => seq($.parameter, repeat(seq(',', $.parameter))),
    parameter: $ => seq($.identifier, $.type_name),

    top_level_statement: $ => /[^#\n]+/,
    doc_block: $ => seq('"""', repeat(/[^\"]+/), '"""'),
    comment: $ => token(seq('#', /.*/)),
    qualified_name: $ => seq($.identifier, repeat(seq('.', $.identifier))),
    type_name: $ => seq($.qualified_name, optional('[]')),
    identifier: $ => /[A-Za-z_][A-Za-z0-9_]*/,
    integer: $ => /[0-9]+/,
    string: $ => /"([^"\\]|\\.)*"/,
  },
});
