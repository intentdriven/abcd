---
schema_version: 1
id: "iss-138"
slug: "payments-protocols-no-position"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "itd-100 terminology crosswalk"
found_at: "docs/reference/terminology.md"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Are agent payment protocols out of abcd's scope by design, so the record closes wontfix?"
---

payments protocols have no recorded position: the itd-100 terminology crosswalk found the record silent on agent-led payments (AP2 at the FIDO Alliance, x402, the two-vendor Agentic Commerce Protocol). The crosswalk row is WATCHING because a REJECTS needs a recorded reason; this capture is where that reason gets decided — likely out-of-scope-by-design for a development-configuration tool, but that is a maintainer call, not an inference. Detector: the crosswalk row cites this id; resolving it means recording the position and updating the row.