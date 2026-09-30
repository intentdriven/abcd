---
schema_version: 1
id: "iss-2609302210137965"
slug: "when-abcd-site-build-adds-missing-interface-labels-to-the"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
remedy: "Carry the labels written with the failure: site.Build wraps an error returned after addMissingLabels wrote in an error that names the file and the labels (unwrapping to the cause, so errors.Is and the message are unchanged), and the CLI names each added label on stderr, as on success, before it reports the error."
---

When abcd site build adds missing interface labels to the repository's ui.json (addMissingLabels, the TG1 ruling) and the build then fails, the error returns an empty Result, so the CLI never names the labels it wrote: the person's file changed and nothing said so, against ADR 2609301720596683's 'the change it takes is visible' (review-tgLabels, minor).
