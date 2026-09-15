---
schema_version: 1
id: "iss-2609011423385217"
slug: "the-release-gate-manifest-s-pinned-inputs-are-stale-and-unde"
severity: "critical"
category: "drift"
source: "agent-finding"
found_during: "the v0.7.0 cut, iss35 full-tier crosscheck"
found_at: ".abcd/development/release-gate/manifest.json"
origin: researcher-authored
production_mode: hand-written
resolution: "The release-gate manifest is held to the tree by TestReleaseGateManifestIsCurrent: the pinned context names all 22 commands/ pages and all 15 agents/ prompts, briefDocs pins every 04-surfaces/ chapter (index included) plus the constraints, internals and delivery chapters where the v0.7.0 findings lived, the context says a surface claim outside the pinned list is in scope, checkerCount equals briefDocs plus surfaces (36), and promptHash is recomputed under the algorithm recovered from the manifest's birth commit (sha256 over the three prompt parts joined by blank lines). The test fails when a command page, agent or surface chapter ships unpinned, when checkerCount disagrees with the lists, or when the prompt is edited without its hash."
impact: fix
---

The release-gate manifest's pinned inputs are stale and under-scope the record
the crosscheck must search. The file is the reproducibility anchor for the
`iss35-brief-surface-crosscheck` gate: every receipt echoes its sha256 as
`manifestHash`, and `checkReceiptGate` refuses a receipt whose hash does not
match. Its own `_comment` states the purpose plainly, that two honest runs of the
same tier mean the same thing because the doc list, the directions, the checker
count and the prompt are fixed here rather than chosen per run.

Fixing the inputs is what makes the runs comparable. It does not make them
correct, and these inputs are not.

## Two defects, the second worse than the first

**The pinned context's content list is stale.** It states that the whole `/abcd:`
surface is "commands under commands/ (ahoy, capture, consult, ingest,
prepare-this-repo, docs, history, launch, memory, version)". That is 10 names.
The tree carries 22 files under `commands/`, so the list omits `abcd`, `banlist`,
`disembark`, `embark`, `guard`, `ideate`, `identity`, `intent`, `lint`, `reading`,
`site` and `update`. All three agents running the v0.7.0 crosscheck reported this
independently and without prompting. One noted that working from the pinned list
alone would have missed more than half the surface.

**The pinned context under-scopes the search, and this is the serious half.** It
directs the run at `.abcd/development/brief/04-surfaces/*.md` and
`05-internals/08-skills.md`. Six of Direction B's thirteen findings live outside
those chapters, in `05-internals/01-agents.md`, `05-internals/03-configuration.md`,
`05-internals/05-prompt-quality.md` and `06-delivery/01-build-sequence.md`. Among
them are all four undocumented `cold-reading-*` agents, the highest-value finding
of the entire pass, and three separate stale agent counts that contradict each
other.

**A run that obeyed the pinned scope literally would have found none of them.**
The v0.7.0 run found them only because the agents were told to treat the shipped
binary as ground truth, which the pinned context's own first line directs, and to
report anything that fell outside the named chapters. That instruction is not in
the manifest. It was supplied at dispatch time, so the depth actually achieved is
not a property the manifest can reproduce.

## Why this is critical rather than major

The gate's value is that a receipt means something specific. Today a receipt can
attest `tier: full` and a matching `manifestHash` while covering materially less
of the record than another run bearing the identical attestation, and nothing in
the receipt distinguishes them. The anchor makes runs comparable to each other
and not to the thing they are meant to measure, so it reproduces a blind spot
instead of catching it.

The v0.7.0 receipt at
`.abcd/work/reviews/4b4076a10f89d4d02da359274dad8994e30cae0e/iss35-brief-surface-crosscheck.json`
records this caveat in its `_reviewProvenance` rather than resting on the hash,
and PROMOTE was given with the caveat disclosed. That is a maintainer decision
about one release, not a fix.

## Also noticed

`checkerCount` is 28, while `briefDocs` holds 24 entries and `surfaces` holds 5,
which is 29 checkers at one per item per direction. The count and the lists
disagree, and nothing reads `checkerCount` to check them against each other.

## Grounds

- pursued: recording this before the fix, per the standing rule against fixing
  ahead of an armed detector; the detector here is the gate itself, and the
  finding is evidence about the gate rather than about any release

- pursued: the anchor must reproduce the search it attests, so the pinned inputs are derived from the tree under a test rather than restated by hand; scope widened by naming the chapters where surface claims live plus an in-scope rule for the rest, which is the exclusion shape the record asked for without a design decision about the whole brief

## Candidate remedies

Not decided; the routing wants a maintainer, and this looks like more than one
record.

- Regenerate the pinned content list from the tree rather than restating it in
  prose, so the manifest cannot describe a surface that has moved. The list is
  derivable from `commands/`, `agents/` and the binary's own command tree, all of
  which `surface.json` already snapshots under a passing gate.
- Widen the search scope to the whole brief, or state the scope as an exclusion
  rather than an enumeration, so a chapter added later is in scope by default
  rather than out of it by omission. Every finding outside the named chapters
  argues for this.
- Reconcile `checkerCount` with the lists, or derive it and drop the field.
- Consider whether the manifest should carry the dispatch-time instruction that
  actually produced this run's depth, since a depth achieved by an unrecorded
  instruction is not reproducible whatever the hash says.

The first two are the substance. The decomposition rule applies before any of it
is filed as an intent: at minimum there is a capability here (derive the pinned
inputs) and a trust rule (what a `manifestHash` is allowed to assert), and those
route to different homes.
