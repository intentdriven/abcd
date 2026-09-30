---
schema_version: 1
id: "iss-25"
slug: "related-sources-frontmatter"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "sources-ingest session 2026-07-08"
deferred_after: "v0.11.1"
deferral_reason: "ruling F owed to the product thinker: no schema admits a related_sources key and nothing reads or writes one, and abcd source add mints no ref_id. Sanctioning the field (public sources by CSL key, confidential ones by an opaque per-source id, resolved through the corpus) is a schema choice with no planned intent. (re-checked at e792a2314 by lane drainDQ3, run A, 2026-09-29)"
remedy: "Waits on ruling F (sanction related_sources): if sanctioned, add an optional related_sources list to the intent, ADR and research-note schemas holding either a CSL-JSON item id from research/references.csl.json or an opaque random per-source id that abcd source add mints at ingest (never a content hash), with record-lint resolving CSL ids against the store and checking only the shape of opaque ids, since CI holds no corpus, proven by a record-lint test per shape; if declined, wontfix the record and keep the corpus-side used_in ledger as the one provenance path."
---

related_sources frontmatter for record-store documents (intents, ADRs, research notes): machine-readable acknowledgment of the sources that informed a document — public sources by CSL key, confidential sources by a random opaque per-source id assigned at ingest (NEVER a content hash: hashing an obtainable document is verifiable by outsiders and breaks confidentiality). Needs schema sanction in record-lint, an id-resolution path via the local corpus, and add-source generating ref_id. Complements the ledger's used_in field (the corpus-side half, already shipped).

## Remedy grounds (2026-09-29)

Why: the CSL store already gives every public source a key, so only the confidential half needs a new id, and a gate that runs in CI cannot resolve a corpus that lives in one account's home, so its check stops at shape. SOTA check: the CSL-JSON data schema requires an id (string or number) on every item (https://github.com/citation-style-language/schema/blob/master/schemas/input/csl-data.json, read 2026-09-29). Rejected: CITATION.cff references (https://citation-file-format.github.io/, read 2026-09-29), which describe the software's own citation rather than per-document provenance and would be a second store beside the CSL one.
