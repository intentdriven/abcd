---
schema_version: 1
id: "iss-2609261811321435"
slug: "the-linearity-fixture-json-unicode-escaped-own-homes-in"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/meter_test.go"
---

The linearity fixture json_unicode_escaped_own_homes in internal/adapter/scanner/meter_test.go is the literal home path, not its unicode-escaped spelling: the u002f escapes were folded back into slashes when the fixture was written, so TestScanLineWorkIsLinear never holds the JSON unicode-escape decode of a dense line of escaped homes to the cost bar its name claims, and the shape duplicates own_home_paths. Detector: the fixture's line carries the six-byte backslash-u002f escape, assembled so no tool can fold it again.
