---
schema_version: 1
id: "iss-25"
slug: "related-sources-frontmatter"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "sources-ingest session 2026-07-08"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (planning owed: related_sources frontmatter (schema sanction, opaque confidential ids, add-source ref_id) has no planned intent). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

related_sources frontmatter for record-store documents (intents, ADRs, research notes): machine-readable acknowledgment of the sources that informed a document — public sources by CSL key, confidential sources by a random opaque per-source id assigned at ingest (NEVER a content hash: hashing an obtainable document is verifiable by outsiders and breaks confidentiality). Needs schema sanction in record-lint, an id-resolution path via the local corpus, and add-source generating ref_id. Complements the ledger's used_in field (the corpus-side half, already shipped).