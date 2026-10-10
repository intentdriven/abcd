---
schema_version: 1
id: "iss-2608211139076040"
slug: "canon-record-id-keyspace-residue-across"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "bughunt-b-round-4"
found_at: "internal/core/lint/deliverystate.go"
resolution: "Routed every record-id key and lookup through canonRecordID: the delivery_state bucket map + changelogIntentCitations, the ScanSpecLinks index id (fixing KnownIntents/IntentSpecID keys), the superseded_by and spec intent-existence/back-link lookups, and the spec_id_unique keyspace. Widened canonRecordID's doc comment to name the whole keyspace. Watched-fail: TestDeliveryStatePaddedIntentFilename, TestSpecIDUniqueZeroPadded, TestSpecLifecyclePaddedIntentResolves."
impact: fix
---

The iss-392 canonRecordID fix canonicalised four record-id-uniqueness scan sites but missed the sibling sites sharing the same keyspace: deliverystate.go keyed the intent bucket map and its CHANGELOG citations on the raw id spelling (so a zero-padded intent filename or citation silently failed the drafts-citation delivery_state gate open), speclinks.go/KnownIntents and the superseded_by (lint.go) and spec intent-existence (lint.go validateSpec) lookups keyed/looked-up raw (false-blocking a padded twin), and spec_id_unique keyed on the raw frontmatter id (fail-open on a padded spc twin, the very hole iss-392 claimed to close for 'all three families'). The playbook's second spine: a pattern fixed at 4 of N sites leaves N-4 latent.