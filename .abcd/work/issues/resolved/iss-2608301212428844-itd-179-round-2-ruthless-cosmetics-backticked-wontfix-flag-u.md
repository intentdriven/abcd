---
schema_version: 1
id: "iss-2608301212428844"
slug: "itd-179-round-2-ruthless-cosmetics-backticked-wontfix-flag-u"
severity: "nitpick"
category: "ux"
source: "user-observation"
found_during: "itd-179-round-2-ruthless"
found_at: "internal/surface/cli/cli.go"
resolution: "the wontfix --grounds usage no longer backticks its placeholder and the reference is regenerated, the derived-grounds redaction is counted once, and the whitespace-only reason is refused by its own cause"
impact: fix
resolved_by:
  intent: "itd-179"
---

itd-179 round-2 ruthless cosmetics: backticked wontfix flag usage, double-counted redaction spans, refusal message names a cause that did not occur

Found by the round-2 adversarial ruthless review of build/itd-179. Three nits,
batched.

1. `internal/surface/cli/cli.go:2655` — backticks inside the wontfix `--grounds`
   usage string make cobra's `UnquoteUsage` take the first backquoted word as
   the flag's value placeholder and strip it from the prose, so
   `abcd capture wontfix --help` prints `--grounds declined` instead of
   `--grounds string`. The same wrong string is committed in the generated
   `docs/reference/cli/commands.md:227`. `promote` and `resolve`, which use
   `groundsFlagUsage`, render correctly. Remedy: drop the backticks and
   regenerate the reference doc.

2. `internal/core/capture/workflow.go:362` with `internal/core/capture/
   grounds.go:63` — a wontfix whose grounds are derived from the reason redacts
   the same operand twice and reports the sum, so one redactable span prints
   "redacted 2 span(s) before writing". Both written fields are correctly
   redacted; only the count over-reports, and it changed from 1 to 2 for
   callers reading it.

3. `internal/core/capture/grounds.go:68-69` — the refusal for a whitespace-only
   wontfix reason says "the reason is empty after redaction", naming a cause
   that did not occur; it previously said "wontfix_reason must be a non-empty
   string". A refusal that misnames its cause sends the operator to the wrong
   remedy.

## Grounds

- pursued: we expect a refusal that misnames its cause and a count that over-reports to cost more operator time than the three lines they take to fix, and a help page printing --grounds declined is the visible end of it
