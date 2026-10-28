---
schema_version: 1
id: "iss-2610141647304982"
slug: "ideate-record-validates-citations-only"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "lab lab-261014142726-9c4d759 (a hand-run of /abcd:ideate on one external write-up, 2026-10-14)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ideate/record.go"
remedy: "Deliver itd-2610141646454347: every record id in every leg is resolved, and a primary source must be a URL, a filed source key or a record id."
---

ideate record validates citations only in the record-grill leg: a verdict citing a non-existent intent id in leg 3 and another in leg 1, with 'a remark heard in passing' as a verified primary source, was recorded with exit 0; in the real run a surviving kill attempt rested on a draft presented as shipped (lab finding F-7).
