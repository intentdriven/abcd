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
---

related_sources frontmatter for record-store documents (intents, ADRs, research notes): machine-readable acknowledgment of the sources that informed a document — public sources by CSL key, confidential sources by a random opaque per-source id assigned at ingest (NEVER a content hash: hashing an obtainable document is verifiable by outsiders and breaks confidentiality). Needs schema sanction in record-lint, an id-resolution path via the local corpus, and add-source generating ref_id. Complements the ledger's used_in field (the corpus-side half, already shipped).