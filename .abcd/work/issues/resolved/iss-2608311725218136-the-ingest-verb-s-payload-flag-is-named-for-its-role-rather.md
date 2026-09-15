---
schema_version: 1
id: "iss-2608311725218136"
slug: "the-ingest-verb-s-payload-flag-is-named-for-its-role-rather"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "walking the operator surface with the maintainer"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/reading.go"
resolution: "The ingest verb's payload flag is now --reading-json, named for what the JSON contains, as its four siblings are. The role-named spelling is gone rather than aliased: an alias would keep the collision with the global --json that the rename exists to remove. The CLI reference page and the release surface export were regenerated, and the plugin page and the brief follow."
impact: breaking
---

The ingest verb's payload flag is named for its role rather than its content, breaking the house idiom its four siblings follow and colliding with the global rendering flag. The repository's output-contract pattern names the flag after what the JSON CONTAINS: review-json, verdict-json twice, changelog-json. This one is output-json, which names the role instead, and role is exactly the ambiguous axis here because the global json flag means the opposite direction of travel: one is how abcd renders its own result, the other is a path to what an agent returned. The verb's own example shows both on one line, which is where a reader trips. Renaming it to reading-json makes it unambiguous about whose output it is and consistent with the four siblings. The window is now: the verb shipped today and has no users, so the change costs a flag rename and a page regeneration, and the cost rises the moment anything scripts against it.

## Grounds

- pursued: renaming the payload flag before the verb has users removes the collision with the global rendering flag at the cost of one regeneration; a script or a page still spelling output-json after this would show the sweep incomplete
