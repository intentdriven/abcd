---
schema_version: 1
id: "iss-2610230410049950"
slug: "the-remedy-text-a-release-hold-finding"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "iss-2610222014550692 round-1 review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/withdrawforge.go"
remedy: "Point a queued pull request's remedy at the dequeue the hold itself performs (or at abcd reserve --admit), never at a bare disable-auto that leaves no record."
resolution: "The release hold's hand remedy for a queued pull request now points at taking it out of the merge queue, or at --admit, never at a bare --disable-auto, and every hand remedy says nothing re-arms what it disarms (internal/core/launch/mergehold.go disarmHint)."
impact: fix
---

The remedy text a release-hold finding gives for a queued pull request, run gh pr merge N --disable-auto, may not take it out of the merge queue, and it disarms the pull request with no withdrawal recorded, so nothing re-arms it after the tag.
