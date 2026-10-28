---
schema_version: 1
id: "iss-2610262033228511"
slug: "05-intent-s-cut-rewrites-every-passed"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 396 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:75"
remedy: "none (filed automatically)"
finding_ref: cfk-219fac169619d87b
---

05-intent's cut rewrites every passed target_release to next, so 09-reflect's list of a cut release's unshipped targeted intents is always empty.

A sequencing impossibility (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:75` — "The cut moves every target it passes — `next`, or a tag at or below the release it cuts — to `next`"
- End B: `.abcd/development/brief/04-surfaces/09-reflect.md:59` — "The planned intents whose `target_release` names the release and which"

05-intent says the cut moves every target at or below the release it cuts to `next`, in the same write as the dated changelog section, which is before the tag exists. 09-reflect runs only on a release tag the repository already holds and lists the planned intents whose `target_release` names that release. By then no planned intent can still name it. So the seed's list, the interview's warning and the write's `unshipped_targets` refusal (with its proceed flag) can never fire as written.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 396.
