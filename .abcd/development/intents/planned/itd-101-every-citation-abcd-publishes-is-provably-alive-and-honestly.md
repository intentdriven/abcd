---
id: itd-101
slug: every-citation-abcd-publishes-is-provably-alive-and-honestly
spec_id: spc-17
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
---

# **Every citation abcd publishes is provably alive and honestly labelled — and the gate that enforces it never flakes.** A cited reference page is only as trustworthy as its links, and links rot silently: pages retitle, URLs redirect, whole platforms announce their own shutdown. abcd splits citation validation across a deterministic gate and an explicit refresh. An offline lint family checks structure and source policy on every commit — footnote markers and definitions in bijection, URLs and DOIs well-formed, aggregator domains refused — with zero network in the gate. `abcd docs cite refresh` does the live fetching on demand and writes a committed baseline: per URL, the final resolved address, when it was last checked, and whether verification was automatic or manual — never how. Sources that block automated fetchers join a manual queue the maintainer clears link by link: printed as a checklist first, later as a generated, disposable checklist page that hands back a receipt file the verb ingests. The lint then enforces the baseline offline — no broken entries, no stale entries, no cited URL without a receipt, no citation whose recorded final address has drifted. The engine is native and dependency-free first; a specialist link checker can slot in later behind the same seam. "My release gate stays deterministic and my citations stay alive," said Kira, an open-source maintainer. "When a source needs human eyes, abcd hands me the exact list to click through and records that I did — not how I did it."

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

**Every citation abcd publishes is provably alive and honestly labelled — and the gate that enforces it never flakes.** A cited reference page is only as trustworthy as its links, and links rot silently: pages retitle, URLs redirect, whole platforms announce their own shutdown. abcd splits citation validation across a deterministic gate and an explicit refresh. An offline lint family checks structure and source policy on every commit — footnote markers and definitions in bijection, URLs and DOIs well-formed, aggregator domains refused — with zero network in the gate. `abcd docs cite refresh` does the live fetching on demand and writes a committed baseline: per URL, the final resolved address, when it was last checked, and whether verification was automatic or manual — never how. Sources that block automated fetchers join a manual queue the maintainer clears link by link: printed as a checklist first, later as a generated, disposable checklist page that hands back a receipt file the verb ingests. The lint then enforces the baseline offline — no broken entries, no stale entries, no cited URL without a receipt, no citation whose recorded final address has drifted. The engine is native and dependency-free first; a specialist link checker can slot in later behind the same seam. "My release gate stays deterministic and my citations stay alive," said Kira, an open-source maintainer. "When a source needs human eyes, abcd hands me the exact list to click through and records that I did — not how I did it."

## Scope Conditions

None stated.

## Acceptance Criteria

- Given `abcd docs lint` runs in the commit gate, when the citations family evaluates, then it uses zero network: footnote structure (marker-definition bijection, and every crosswalk table row carrying at least one footnote), URL/DOI syntax, and source-domain policy come from committed config and the committed baseline.
- Given `abcd docs cite refresh` runs, when it writes the baseline, then each URL entry records the final resolved address, when it was checked, and whether verification was automatic or manual (with its date) — never how.
- Given a baseline entry older than 180 days, when docs lint runs, then it warns; given one older than 365 days, when the release gate runs, then it blocks; human-verified entries age on the same clock and re-enter the manual queue when stale.
- Given a source that blocks automated fetchers, when the maintainer clears the manual queue, then a printed checklist plus a confirm verb writes the dated receipt; the later generated checklist page hands back a receipt file the same verb ingests — one receipt schema for both rungs.
- Given a specialist link checker is adopted later, then it slots behind the refresh seam as an adapter without changing gate semantics — internal basic first, SOTA dependency later.

## Open Questions

- The generated checklist page's receipt-file format details (settled only as: same schema as the confirm verb writes).
- Where the scheduled-CI refresh wrapper lands when it earns its own sign-off (explicitly out of this intent's scope, per the 2026-07-27 grill).

## Grill Settlements (2026-07-27)

- Refresh is manual now, surfaced by ahoy status and release-preflight nagging; a scheduled-CI wrapper is a later, separately signed-off change.
- Staleness policy: warn at 180 days in docs lint; blocker at 365 days at the release gate only — commits are never calendar-blocked.
- The row-has-footnote structural rule deferred from spc-15 is homed here (DECISIONS 2026-07-27): implemented in this intent or not at all.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
