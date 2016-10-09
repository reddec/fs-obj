# File system to object generator

This is simple utility that generates `struct`s and methods for objects on file system.
Think about it like lightweight ORM on fs. It's ready for `go:generate`

Now little formal description

### Install

Same as any other GoLang applications:

`go install github.com/reddec/fs-obj`


### Usage

```
fs-obj [flags...] <URLs,...>

Flags:

  -o string
        Output folder name (default "./dummy")
  -p string
        Package name (default "dummy")
  -r string
        Root folder name (default "data")
```

URL in this terms is a description of FS object and have those grammar:

`(/[:]SECTION)+ [-> CLASS]`

Where `SECTION` is a single part of path to target file. If `SECTION` starts with `:`, it means to be dynamic (like a `:param` in URL routing)

Where `CLASS` is a optional name of structure which will be coded/encoded by JSON codec.

### Let's show some examples

Command:

`fs-obj -o auth -p auth '/groups/:group/user' '/rules/common/:rule->Rule' '/rules/personal/:user/rule->Rule'`

Will generates this file tree:

```
auth
├── fsdb_data_groups_group_user_record.go
├── fsdb_data_rules_common_rule_rule.go
└── fsdb_data_rules_personal_user_rule_rule.go
```

And generates factory method in package api: `NewData()`
