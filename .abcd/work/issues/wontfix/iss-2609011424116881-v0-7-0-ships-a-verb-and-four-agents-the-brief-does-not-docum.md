---
schema_version: 1
id: "iss-2609011424116881"
slug: "v0-7-0-ships-a-verb-and-four-agents-the-brief-does-not-docum"
severity: "major"
category: "documentation"
source: "agent-finding"
found_during: "the v0.7.0 cut, iss35 full-tier crosscheck"
found_at: ".abcd/development/brief/05-internals/08-skills.md"
origin: researcher-authored
production_mode: hand-written
wontfix_reason: "duplicate of iss-2609011424033149: the same fifteen release-introduced findings captured twice from one receipt; this record carries the fuller body, and the resolution with fixing-commit provenance is recorded on the earlier id"
---

v0.7.0 ships a verb and four agents the brief does not document anywhere. These
are the fifteen crosscheck findings that are this release's own doing, kept apart
from the twenty-one pre-existing ones in
[[iss-2609011424113262]] because they are a different decision: this is the
release outrunning its own record, not debt it inherited.

Every one is recorded with its evidence in the v0.7.0 receipt at
`.abcd/work/reviews/4b4076a10f89d4d02da359274dad8994e30cae0e/iss35-brief-surface-crosscheck.json`,
dispositioned `deferred-release-introduced`.

## The four cold-reading agents

`agents/cold-reading-widening.md`, `-entailment.md`, `-comparative.md` and
`-detection.md` ship with full frontmatter (`prompt_version`,
`reads_untrusted_input`, `capability_scope`, `position`, `regime`, `color`) and an
injection-canary fixture each. The shipped binary names them in bare `abcd
reading` output. `grep -rn "cold-reading-widening" .abcd/development/brief/`
returns nothing, and the same holds for all four.

They are absent from the 16-agent roster table (`05-internals/01-agents.md:9-24`),
from "Shipped agents outside the design roster" (`:30-38`), and from the `agents/`
tree at `05-internals/03-configuration.md:363-366`.

## Four agent counts, none of them right, two contradicting each other

- `05-internals/01-agents.md:3` says eleven prompt files ship. Fifteen do.
- `05-internals/01-agents.md:28` says seven ship outside the design roster. Eleven do.
- `05-internals/05-prompt-quality.md:3` repeats the eleven.
- `05-internals/03-configuration.md:362-366` says ten and lists ten by name,
  omitting `scribe.md` (which `01-agents.md:38` documents) and all four
  cold-reading prompts. It contradicts `01-agents.md:3` directly.

Two frontmatter carrier claims are stale with them: `color` is described as
carried by five prompts (nine carry it) and `capability_scope` by eleven (fifteen
carry it). Two shipped frontmatter fields, `position` and `regime`, have no row in
the frontmatter table and are not in its deliberately-omitted list.

## The reading verb

- `05-internals/08-skills.md:19` says abcd ships seventeen binary-backed top-level
  commands and enumerates seventeen without `reading`. Eighteen ship: `reading`
  has a Go verb and a `commands/reading.md`.
- The same line says "the three newest parents carry small trees" and then lists
  four, omitting `reading`, which is the actually-newest and carries `assemble`
  and `ingest`.
- `04-surfaces/08-abcd.md:79-83,87` says the `/abcd:<verb>` surface is twenty verb
  files and omits `reading`. Twenty-one ship.
- Three separate bare-status-board enumerations omit `reading` and `site`, both of
  which render boards: `05-internals/08-skills.md:17`,
  `04-surfaces/10-docs.md:40-43`, `04-surfaces/12-version.md:26-28`. The last two
  maintain the same list independently, which is why it drifted twice.
- `04-surfaces/23-reading.md:31` has a dangling `[adr-58]` reference-style link
  with no definition, so it renders as literal text.

`04-surfaces/23-reading.md` is otherwise **accurate**, and was given targeted
scrutiny: three required operands, comparative refusing rather than assembling,
`--reading-json`, the closed position set, all six refusals exiting 2, the run-id
mint and the bundle carrying no repository path all verified against the built
binary. The chapter that describes the new verb is right; the chapters that count
things are wrong.

## Grounds

- deferred: not fixed in the v0.7.0 cut, because the fix is prose across seven
  brief files and the release was already two rounds deep in gate corrections;
  deferring was the maintainer's recorded decision at PROMOTE, with the split
  from the pre-existing drift recorded so this half is not lost inside it

- declined: duplicate of iss-2609011424033149: the same fifteen release-introduced findings captured twice from one receipt; this record carries the fuller body, and the resolution with fixing-commit provenance is recorded on the earlier id

## Scope for the follow-up

This is the half that should not wait. A release whose record cannot say what it
ships is the condition the crosscheck gate exists to detect, and it detected it.
The counts are the cheap part; the four undocumented agents are the substance,
because an agent with no brief row has no recorded capability scope, no stated
threat model and no roster entry to review it against.
