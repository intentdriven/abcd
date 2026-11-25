---
schema_version: 1
id: "iss-2611252207514940"
slug: "refs-line-mixing-families-is-refused"
severity: "nitpick"
category: "ux"
source: "agent-observation"
found_during: "records commit of 25 November"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/check-issue-resolution.sh"
remedy: "Accept one Refs: line carrying ids of several families, since a reader sees the same declaration; or, if the split is wanted, say so on the committing page and in the ATTRIBUTION rule beside the trailer forms."
---

In a commit message, a Refs: line naming an issue beside an id of another family (Refs: itd-N, iss-M) is refused as no declaration (RS004), though the same line with ids of one family is accepted; the refusal says so, but nothing in the commit pages or the attribution rule tells a writer before. Met on 2026-11-25.
