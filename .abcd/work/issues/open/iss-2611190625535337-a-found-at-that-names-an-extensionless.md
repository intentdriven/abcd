---
schema_version: 1
id: "iss-2611190625535337"
slug: "a-found-at-that-names-an-extensionless"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous run of 15 Nov 2026, building iss-2611190439574731"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/foundat.go"
remedy: "Count a found_at that names a file existing in the checkout as a path even with no slash and no extension, and keep capture's own check, the build's reservation and the drain on that one verdict."
refines: [iss-2611101316419989]
---

A found_at that names an extensionless file at the repository root, such as Makefile, reads as a place in words rather than a path: foundAtPath takes a value as a path only when it holds a slash or ends in a file extension. So abcd build refuses an issue found at Makefile with no found_at path to reserve it with, and the drain hands such an issue back for the same reason, though the record names a real file of the checkout.
