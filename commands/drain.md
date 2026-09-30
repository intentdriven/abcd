---
name: drain
description: "Sort open issues by this repository's own drain rule, naming each loosened floor: Writes nothing; refuses without the rule's record, or without --dry-run."
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

Which issues a drain may take alone is **this repository's own decision**: an
accepted decision record in `.abcd/development/decisions/adrs/` whose
frontmatter carries four fields. abcd's strict baseline, which
`abcd ahoy install` offers to write, is:

```yaml
drain_categories: [tech-debt, documentation, inconsistency, drift, bug, ux]
drain_severities: [nitpick, minor]
drain_security: handback
drain_remedy: required
```

A repository's record may narrow those lists, and may **loosen abcd's floors**:
list `major` or `critical` in `drain_severities`, or set `drain_security: take`.
It cannot widen the categories past that fixable set, and `drain_remedy` is
always `required`.

An open issue is **eligible** when all of these hold, read from its fields
alone:

- no record in its `blocked_by` is still open;
- its category is one the rule takes;
- its severity is one the rule takes;
- it carries a `remedy:` (`abcd capture --remedy "<fix>"`, which every new
  issue carries, or `abcd capture remedy <iss-N> "<fix>"` onto an open one; a
  record carrying only the older `suggested_fix:` reads that as its remedy),
  the remedy is not `none (filed automatically)`, the value abcd's automatic
  filers write when they have no fix, and it does not open "Waits on";
- it carries no deferral that is live at the checkout's newest release tag, and
  none at all when the checkout holds no release tag.

The rules are asked in a fixed order, and the first that excludes an issue
decides its one disposition:

| `outcome` | `rule` | Meaning |
| --- | --- | --- |
| `skipped` | `blocked` | an open record blocks it; `blockers` names them |
| `handback` | `security` | category `security`, a person's unless the rule sets `drain_security: take` |
| `handback` | `category` | a category the rule does not take |
| `handback` | `severity` | a severity the rule does not take (`major` and `critical` under the baseline) |
| `handback` | `waits-on-ruling` | its remedy opens "Waits on": the fix waits on a ruling a person has not given |
| `handback` | `deferred` | its `deferred_after` names the current anchor tag: a person carried it past this release; or the checkout holds no release tag (a shallow clone fetches none), so whether any deferral is live is unknown and the reason names `git fetch --tags` |
| `ineligible` | `remedy` | no remedy (a record filed before the remedy was required), or `none (filed automatically)` from an automatic filer; ineligible until a person writes one with `abcd capture remedy`, which the reason names |
| `unreadable` | `unreadable` | the ledger reader refuses the record; the reason names why |
| `eligible` | `fields` | every field rule passes |

An eligible issue is not yet promised a lane: the host judgement over its
remedy (a user-visible or trust-boundary change hands it back) does not run in
a dry run, and it can only ever hand an issue back.

## The order

Eligible issues come first, in the order a drain takes them: by category
`tech-debt`, `documentation`, `inconsistency`, `drift`, `bug`, `ux`, then
`security` when the rule takes it; then by severity, `nitpick` first; then
oldest first. The payload's `order` states the rule. Every other open issue
follows, by id.

## Loosened floors

When the repository's record loosens a floor, the text output prints a
`LOOSENED` block naming each one (`severity major`, `severity critical`,
`security`), stderr carries a warning naming them in both modes, and the
payload's `loosened` lists them. **Tell the user every loosened floor first,
before anything else in the plan**: a loosened rule lets a drain take issues
abcd's baseline hands to a person, and the record that loosened it is a change
a person should have reviewed.

## The payload

`dry_run` is `true`; `record` is the repository's decision record the rule is
read from; `rule` is that record's rule (`record`, `path`, `categories`,
`severities`, `security`, `remedy`, `loosened`); `loosened` lists every floor
it loosens (empty when none); `anchor` is the release tag a live deferral names,
present when an open record carries a deferral; `anchor_unknown` is `true` when
an open record carries a deferral and the checkout holds no release tag; `order` is the ordering rule;
`dispositions` holds one entry per open issue (`id`, `path`, `severity`,
`category`, `outcome`, `rule`, `reason`, and `blockers` when skipped); `counts`
totals them by outcome; `ledger` names the checkout and branch read.

Tell the user any loosened floors, then the counts, then the eligible issues in
order, then the others grouped by outcome with their reasons. For an
`ineligible` issue, say that a person writing a remedy with
`abcd capture remedy <iss-N> "<fix>"` is what makes it a candidate, and name the
ones an automatic filer wrote apart, since their reason says so. For a
`waits-on-ruling` or `deferred` hand-back, say which ruling or release it waits
on. Do not act on the list: a hand-back is a person's decision.

## Refusals

- Without the repository's own record of the rule, the dry run and the bare
  verb refuse (exit 2, nothing written), naming how to add it: run
  `abcd ahoy install` at a terminal and accept the offer, or give an accepted
  decision record the four `drain_` fields. A record carrying the fields but
  proposed or superseded is named. Relay this; do not write the record for the
  user.
- A malformed record (a field missing or misspelt, any frontmatter key stated
  twice, an `id` its file name does not give it, or a value the field does not
  take) refuses, naming the record and the field; two accepted records carrying
  the fields refuse, naming both. A decision store or record that cannot be read
  safely (a store or record that is a symlink, wherever it points, or a record
  past the size cap) refuses. Every one of these
  exits 2 with nothing written, on the dry run and the bare verb alike.
- Without `--dry-run` the verb refuses to start (exit 2, nothing written): the
  issue-keyed lane a drain hands each issue to is not built. The refusal names
  the rule's record, every floor it loosens, and the dry run.
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
