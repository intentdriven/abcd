---
id: dcn-2608260000000001
slug: bughunt-round-7-state-368-hardened-the
date: 2026-08-26
---
2026-08-26 — bughunt round 7 (state #368): hardened the new issue-resolution
gate scripts. check-issue-resolution.sh now cd's to the repo root (RS003 no
longer passes vacuously from a subdirectory), extracts iss-ids non-fatally (a
non-record file last in the diff no longer silently aborts RS001/RS002 under
set -e), and RS002 reads each changed record's frontmatter via a helper shared
with RS003 instead of the raw diff (a body-level commit: example is no longer
reachability-checked); check-issue-resolution-cases.sh gained the
check-attribution-cases hermetic git-env scrub (iss-28/iss-313 sibling authored
four days after that fix). iss-2608261040378346, iss-2608261041020040. Doc
corrections: the 05-internals and 04-surfaces indexes no longer call the shipped
site check gates / explorer / deploy design targets (10-site.md is the correct
side; adr-48), brief/README nine→ten internals chapters, capture.md --shipped-in
named as derivation-time not resolve-time validation, CONTEXT surface_coverage
bullet names the sub-verb CLI-tree pass. iss-2608261041027043,
iss-2608261041029596, iss-2608261041027268, iss-2608261041020419. Recorded not
fixed (autonomous-scope limits): release.yml feeds an unvalidated tag into make
build's -ldflags shell recipe — a shape-check-parity gap vs site.yml,
insider-bounded, deferred because CI cannot exercise the release pipeline
(iss-2608261041218890); the principle spec-moves-with-the-surface sits unpromoted
while its named surface_coverage gate is armed at blocker — promotion is a
governance act for the maintainer (iss-2608261041210476). Refuted: a resolved_by
bare-null "regression" (the shape is unwritable and the new rejection removes a
phantom empty-pointer); the release.yml verify job "not mirroring ci" (a
deliberate, gate_lockstep-checked subset).
