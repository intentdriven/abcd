---
schema_version: 1
id: "iss-2610121653499543"
slug: "abcd-capture-list-json-and-the-other"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/workflow.go"
remedy: "Mask a record's text where a capture surface composes its payload (termsafe.SanitizeBlock for the body, termsafe.Sanitize for each frontmatter string), never where the record is read: resolve, defer, promote and every other rewrite round-trip the stored text, and masking at read would write the mask into the record. Test with a record whose body and remedy carry U+202E, U+009B and U+034F, asserting nothing raw on capture list --json and the record file unchanged after a resolve."
---

abcd capture list --json, and the other capture surfaces that emit an issue whole, echo a record's stored text raw: its body and its frontmatter strings (remedy, found_during, resolution, wontfix_reason and the rest). capture writes a body with its hidden runes percent-encoded, but the validator holds no stored string to a rune set, so a hand-edited or repository-shipped record carrying a control, bidi or zero-width rune puts it on the stream. Found while masking the skipped-record list for iss-2610121542258778; a test planting U+202E and U+034F in a record body sees them raw in capture list --json.

Security scope (abcd-fixes-what-it-adds): the route does not exist without abcd. The text is echoed by abcd's own --json surface, so abcd added the route and fixes it at the boundary the text enters by; a fix changes nothing about how the repository works.
