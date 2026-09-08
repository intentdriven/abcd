---
schema_version: 1
id: "iss-2609081940550352"
slug: "assisted-by-vendor-prefix-demotes-a-human-author"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "bughunt-triage"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/contributors.go"
---

LoadAuthorship puts every non-None Assisted-by value into ByModel and registers its colon-prefix as a vendor, then routes any shortlog author whose name equals a vendor key into the Bots and tools row. A conformant trailer whose vendor token is an existing single-token git author name therefore moves that human author whole shortlog count into Bots on /contributors/. The match is exact and case-sensitive. A second, free-form Assisted-by line also publishes as a bar label, because scripts/check-attribution.sh is a presence check (grep -Eq TRAILER_RE) rather than an every-line-conforms check, and underAttributionEscape skips banned tokens on empty data-src spans. Display only, and the trailer stays review-visible. The vendors[au.Name] clause must not simply be dropped: TestAuthorshipSeparatesToolsFromPeople pins the pre-policy tool-author case it implements (iss-2608220150157501). Fix: demote to Bots on a structural machine-identity signal (a [bot] suffix, or a vendor-name match AND a noreply/bot address) so a vendor prefix cannot reclassify a human while Claude at a noreply address and the fixture Assistant author stay in Bots; chart only trailer-shaped values (the attribution grammar including the optional 1m suffix) and clip or drop free-form extras rather than publishing them as labels. Detector: a fixture author named like a human of record, plus an Assisted-by trailer whose vendor is that same name, stays in Humans; TestAuthorshipSeparatesToolsFromPeople still puts Assistant and depbot[bot] in Bots; a free-form extra trailer does not appear as a ByModel label. Reported as GitHub issue 623 against ec7f40d6.
