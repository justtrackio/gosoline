# CLI flags

Declare boolean switches with `Kind: cli.FlagKindBool`:

```go
cli.Flag{Long: "worktrees", Short: "w", Kind: cli.FlagKindBool}
```

A bare boolean switch is always `true` and never consumes the next argument.
Both `clone --worktrees topic workspace` and `--worktrees clone topic workspace`
keep the command and positional arguments intact. Use `--worktrees=false` or
`-w=false` for an explicit value. Absent boolean flags default to `false` unless
`Default` specifies a different boolean value. Repeated switches use the last value.
Invalid explicit values and defaults produce an error during blueprint creation.

`Cli.Run` uses command, group, and global flag definitions when parsing. Aliases
can have different kinds in different commands. String and list flags retain
value-taking behavior. `--` ends flag parsing, and grouped short switches remain
supported.

For parsing without running an application, use `NewInputWithRouter(args, router,
globalFlags...)`, then `NewBlueprint(router, input, globalFlags...)`. For a flat
set of flag definitions, use `NewInputWithArgs(args, flags...)`. Calls without flag
definitions preserve the existing string parsing behavior; built-in help switches
are always boolean.
