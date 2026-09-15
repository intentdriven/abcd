---
schema_version: 1
id: "iss-2608300258510460"
slug: "own-artefact-refusal-is-extension-and-parse-shaped"
severity: "major"
category: "security"
source: "impl-review"
found_during: "itd-183 second-round security review, 2026-08-30"
found_at: "internal/core/reading/assemble.go (refuseOwnArtefact)"
resolution: "The refusal keyed on a .json name and a successful top-level unmarshal, which made it a spelling check rather than a recognition: the same artefact committed as .yaml or .toml, behind a byte-order mark, wrapped in another object, wrapped in an array, or carrying a duplicate key was admitted whole. What identifies the artefact is the tag it carries, so the tag is now looked for in the bytes of every admitted file before anything is parsed. The parse stays on behind it and earns its keep on the one shape a byte scan cannot see, a tag spelled with a JSON unicode escape."
impact: fix
---

The own-artefact refusal is extension- and parse-shaped: it keys on a .json extension and a successful top-level unmarshal, so a prior manifest or bundle committed as .yaml or .toml, or as .json with a BOM, wrapped in another object, wrapped in an array, or carrying a duplicate _type key, is admitted whole while the new manifest asserts the exclusion. Make the check content-signed: refuse any admitted file whose bytes contain either artefact _type tag string before any parse, regardless of extension, keeping the parse as a secondary check.
