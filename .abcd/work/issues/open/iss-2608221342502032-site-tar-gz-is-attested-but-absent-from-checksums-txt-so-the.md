---
schema_version: 1
id: "iss-2608221342502032"
slug: "site-tar-gz-is-attested-but-absent-from-checksums-txt-so-the"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "agent-finding"
found_at: ".github/workflows/site.yml"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Should checksums.txt cover site.tar.gz, or is the manifest binaries-only by design?"
---

site.tar.gz is attested but absent from checksums.txt, so the release manifest no longer covers every asset; wants a deliberate decision rather than a silent gap