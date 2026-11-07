---
schema_version: 1
id: "iss-2611042320121086"
slug: "the-guard-reads-some-forge-calls-as-a"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "coordinating the build lanes, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/varpath.go"
remedy: "Add the missing REST routes to forgeOtherRoutes and -p and --preview to the gh api ref hazards' value_flags, each with a known_good fixture in the registry."
---

The guard reads some forge calls that touch no ref as a PATCH on a ref. Since the history hazards landed in #1394, gh api -X PATCH on repos/$O/$R/properties/values with --input is refused by gh-api-ref-input, because forgeOtherRoutes does not list that route, so the variable path can still be a ref; and a -p "$P" value reads as a possible ref path, because the hazards' value_flags omit -p and --preview. Spelling the path or the flag out avoids both.
