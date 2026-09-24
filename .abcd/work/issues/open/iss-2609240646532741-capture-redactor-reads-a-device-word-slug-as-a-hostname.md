---
schema_version: 1
id: "iss-2609240646532741"
slug: "capture-redactor-reads-a-device-word-slug-as-a-hostname"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture"
---

The store-before-commit redactor on the capture path rewrites a hyphenated prose slug that ends in a device word as `[redacted-hostname]`: the net_device_hostname rule matches any hyphenated token ending in a word such as laptop, macbook or nas, although the author named a page, not a machine. Observed on 2026-09-23 in iss-2609230851376499: its reproduction step names the memory page slug it ingested, and the stored record reads "slug [redacted-hostname]", so the reproduction cannot be followed from the record's own text. #663 fixed the same false positive on the memory lint's read side by judging page back-links by the page-name rule; the capture path applies the free-text rule to every token and has no such distinction. Wanted: the capture path spares a token the author set in code quotes, or the rule asks for a shape only a host name takes; either change carries a fixture with a slug of that shape.
