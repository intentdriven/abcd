---
schema_version: 1
id: "iss-2608311517547712"
slug: "the-provenance-field-is-a-signature-free-channel-through-the"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "itd-185 fidelity audit rcp-fe3450ca55ff"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/ingest_regime.go"
resolution: "The semantic signatures now read every text value an item carries, the envelope pattern included, so the registry's own phrasing placed in the provenance field is seen like any other. An ordinary pattern still raises nothing, and the signature text function says in terms why the field is read: it is untrusted text the reading chose, every item at every regime must carry it, and it lands in a committed record."
impact: fix
resolved_by:
  intent: "itd-185"
  spec: "spc-63"
---

The provenance field is a signature-free channel through the supply-regime gate. The semantic detectors read the body text only, and the envelope pattern field is excluded from it, so an item whose pattern carries the registry's exact phrasing lands with exit 0 at every regime: a registrative pattern reading that the fix is to rewrite the charter, an explicative one carrying a disposition, an evaluative one recommending a candidate. No byte is substituted and no phrasing is novel; the detector simply does not read that field. By the intent's own test, which puts the registry's own phrasing on the defect side and only novel phrasing on the residue side, this is a defect and a wider one than the confusables class that IS disclosed. It is also the field every item must carry, so the channel is always available.

## Grounds

- pursued: a detector's reach is decided by where untrusted text can arrive, not by what a field is nominally for, and the one field every item must carry is the widest channel there is; an ordinary pattern flagged, or the registry's phrasing still landing unremarked in that field, would show this wrong
