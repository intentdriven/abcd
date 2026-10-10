---
schema_version: 1
id: "iss-348"
slug: "the-root-command-s-use-and-help-omit-the"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-round-2"
found_at: "internal/surface/cli/cli.go"
resolution: "root Use declares the record-id positional and a Long explains bare-vs-id dispatch; reference regenerated"
impact: fix
---

the root command's Use and help omit the shipped record-id dispatch, so the generated CLI reference states Usage: abcd with no positional and abcd --help never mentions that abcd iss-N describes a record
## Evidence

- `internal/surface/cli/cli.go:179-194`: root `Use: "abcd"`, no `Long`; `Args` accepts one `record.IDRe` positional and `RunE` dispatches through `record.Describe` (spc-26, shipped v0.6.0). The generated reference derives its usage line from `Use` (`internal/surface/cli/reference.go:68`), so `docs/reference/cli/commands.md:16` states `Usage: abcd` and `abcd --help` never mentions the dispatch.
- Every other command with a positional declares it (`rules [domain]`, `intent [text]`, `capture [text]`, `audit [<itd-N>]`, `probe [repo]`, `promote <iss-N>`); root is the sole exception. Distinct from iss-304 b-4 (Cobra completion/help sections).
- Refuter verdict: CONFIRMED substantive (discoverability). Fix verified in a scratch copy: `Use`+`Long` change regenerates cleanly, drift test and spc-26 golden tests green.
