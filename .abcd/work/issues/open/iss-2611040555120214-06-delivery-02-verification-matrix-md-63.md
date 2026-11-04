---
schema_version: 1
id: "iss-2611040555120214"
slug: "06-delivery-02-verification-matrix-md-63"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md"
remedy: "Add --remedy to the row's form and drop severity from what it reports."
---

06-delivery/02-verification-matrix.md:63 says abcd capture "<text>" writes an issue and reports the id, slug and severity. That form is refused with exit 2 because --remedy is required, and nothing is captured. With --remedy, the text output gives id, status and path, and the --json result carries id, slug, path, status, uncommitted, no_location, match and ledger; neither reports the severity.
