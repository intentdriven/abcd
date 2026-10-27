---
schema_version: 1
id: "iss-2610262032258227"
slug: "reflect-s-set-of-intents-a-release"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 265 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/09-reflect.md:49"
remedy: "none (filed automatically)"
finding_ref: cfk-c831bcb866195ca5
---

Reflect's set of intents a release shipped follows shipped_in both ways; the cut only drops records whose shipped_in an earlier tag reaches

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/09-reflect.md:49` — "less any whose `shipped_in` names another release, plus any in `shipped/` now whose `shipped_in` names this one"
- End B: `.abcd/development/brief/04-surfaces/04-launch.md:526` — "leaving out an entering record whose `shipped_in:` names a release tag the previous tag already reaches, since that release carried its work; a value that names no such tag keeps the record in"

Launch gathers the records that entered shipped/ since the previous tag, leaving out only one whose shipped_in names a tag the previous tag already reaches and keeping one whose value names no such tag. Reflect takes those that reached shipped/ in the window, less any whose shipped_in names another release, plus any in shipped/ now whose shipped_in names this one. A record stamped for a later release is in the cut but out of reflect's set, and one that entered earlier but is stamped for this release is in reflect's set but not in the cut, so the retrospective's intent table and the changelog section it links can disagree.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 265.
