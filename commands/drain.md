---
name: drain
description: "Sort the open issues by the drain's field rule, eligible first in drain order: Writes nothing; refuses to start without --dry-run, as the run is not built."
block: agents
---

# `/abcd:drain`

Show what a drain of the open issue ledger would do: which issues a machine may
fix alone, in the order it would take them, and what happens to every other
open issue. The drain run itself is not built yet, so this page covers its dry
run, which performs **zero writes**.

Run:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" drain --dry-run --json
```

## The rule

An open issue is **eligible** when all of these hold, read from its fields
alone:

- no record in its `blocked_by` is still open;
- its category is in the fixable set: `tech-debt`, `documentation`,
  `inconsistency`, `drift`, `bug`, `ux`;
- its severity is `nitpick` or `minor`;
- it carries a `remedy:` (captured with `abcd capture --remedy "<fix>"`; a
  record carrying only the older `suggested_fix:` reads that as its remedy).

The rule is a recorded decision, and the payload's `record` names it. The
rules are asked in a fixed order, and the first that excludes an issue decides
its one disposition:

| `outcome` | `rule` | Meaning |
| --- | --- | --- |
| `skipped` | `blocked` | an open record blocks it; `blockers` names them |
| `handback` | `security` | category `security` is always a person's |
| `handback` | `category` | a category outside the fixable set (`process`, `observation`, `architectural-insight`, `future-work-seed`, `lapse`) |
| `handback` | `severity` | severity `major` or `critical` |
| `ineligible` | `remedy` | no remedy; ineligible until someone adds one |
| `unreadable` | `unreadable` | the ledger reader refuses the record; the reason names why |
| `eligible` | `fields` | every field rule passes |

An eligible issue is not yet promised a lane: the host judgement over its
remedy (a user-visible or trust-boundary change hands it back) does not run in
a dry run, and it can only ever hand an issue back.

## The order

Eligible issues come first, in the order a drain takes them: by category
`tech-debt`, `documentation`, `inconsistency`, `drift`, `bug`, `ux`; then
`nitpick` before `minor`; then oldest first. The payload's `order` states the
rule. Every other open issue follows, by id.

## The payload

`dry_run` is `true`; `record` is the decision record the rule is stated in;
`order` is the ordering rule; `dispositions` holds one entry per open issue
(`id`, `path`, `severity`, `category`, `outcome`, `rule`, `reason`, and
`blockers` when skipped); `counts` totals them by outcome; `ledger` names the
checkout and branch read.

Tell the user the counts, then the eligible issues in order, then the others
grouped by outcome with their reasons. For an `ineligible` issue, say that
adding a remedy is what makes it a candidate. Do not act on the list: a
hand-back is a person's decision.

## Refusals

- Without `--dry-run` the verb refuses to start (exit 2, nothing read or
  written): the issue-keyed lane a drain hands each issue to is not built. The
  refusal names what is missing and points at the dry run. It would also refuse
  naming the decision record it needs, were the rule unrecorded.
- Outside a checkout, or on a ledger holding one id in two status folders, it
  refuses (exit 2) as every capture verb does.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.
