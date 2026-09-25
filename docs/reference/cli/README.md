# CLI reference

The full command reference lives in [`commands.md`](commands.md): every
user-facing `abcd` command with its sentence, usage line, and flags. The
sentence says what the command does, what it writes, and when it refuses, and
it is the same line the command's `--help` opens with and its command list
shows.

That page is generated from the Cobra command tree in `internal/surface/cli`, so
it always matches the binary. A drift test regenerates the tree on every `go test`
run and fails the build whenever the committed page and the tree disagree, so the
reference stays in step with the code.

To refresh the page after changing a command, run:

```bash
go generate ./internal/surface/cli
```

For interactive help, the binary also documents itself: `abcd --help` and
`abcd <verb> --help` (e.g. `abcd disembark --help`).
