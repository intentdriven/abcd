---
id: spc-2611041649558539
slug: when-an-agent-goes-wrong-abcd-tells-it
intent: itd-2611041636196213
origin: researcher-authored
production_mode: hand-written
---
# when-an-agent-goes-wrong-abcd-tells-it

## Summary

One change holds the prompt hook under the host limit and splits the shell rules
into the ones sent ahead of time and the ones a check carries. It delivers
itd-2611041636196213 and resolves iss-2611041419354106. Rule words do not change:
that review is a later change.

**The budget.** `InjectionBudgetBytes` becomes the whole output of one prompt: the
host cap of 10,000 characters less a 500 margin, the figures the session-start hook
already uses (bytes never undercount characters). The router reads the status and
away blocks first, so the rules and notes get what those leave. Rules leave a 2,500
byte floor for notes, one note at the longest body a note may have; notes take what
the rules actually left, and a turn no note comes back on gives the floor back to the
rules. `rules --prompt` runs the same path with nothing reserved.

**Overflow.** `renderWithinBudget` no longer stops at the first domain that does not
fit. Room goes up a ladder: every matched domain gets its pointer (heading and a line
with the command), then the cheapest widen to the existing `renderShrunk` (heading,
first rule, that line), then the cheapest of those to full, while room lasts. Either
short form is terminal: the
ledger records it as sent, as it already does in a bracket that shrinks, so there is
no second sending and no state for what remains. Rules still come back in full
whenever they fit, so the ruling on spc-2610111627213138 stands. A domain with no
room even for its pointer is left out and tried again; only that case carries the
truncation notice, and its wording stops blaming a file as bloated or hostile, since
routine overflow is not an attack.

**Notes.** The fresh row gains a line bound, so a note or page that does not fit
comes back as one line, the same form the other rows already send, in place of
being left out or set aside as too large.

**The two lists.** `Domain` gains `Checked`, a generated list no rule file can
declare (`json:"-"`), which only SHELL fills. With the guard on, `Rules` holds one
line saying a guard judges each command, then the warned lessons; `Checked` holds the
refused lessons and the three lessons with no entry of their own (script reading,
variable-built `-c`, unprovable delete). With the guard off, every lesson stays in
`Rules` as today and no line claims a guard. The signature and the injected block
read `Rules` only. `abcd rules`, bare or scoped, renders both lists, and `--json`
adds `checked`, `rules_bytes` and `checked_bytes` for every domain.

**What stays.** The override format, the recall words, the star command, dormant,
the kill switch and the override notes. An override of `{"SHELL":{"rules":[]}}` is
dropped with its note as now; since the steering line is generated, `rules` is never
empty unless an override empties it, so the drop logic needs no second list.

**Doc changes.** The AGENTS.md paragraph on SHELL, the configuration chapter of the
brief, the guard surface page, `commands/guard.md`, the reference page and the
`rules` help text stop saying every registry entry is sent ahead of time.

## Footprint

- packages: internal/core/rules, internal/core/guard, internal/core/recall, internal/surface/cli, commands, docs, AGENTS.md, .abcd/development/brief
- tests: internal/core/rules, internal/core/guard, internal/core/recall, internal/surface/cli

## Steps

_No steps listed: the spec is built as one step. To split it, list the steps in order as `1. <title>`, each with `- packages:` and `- tests:` indented beneath it; `- landed: <pull request or commit>` marks a step that has landed._
