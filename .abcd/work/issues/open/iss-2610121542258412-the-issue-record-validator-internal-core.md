---
schema_version: 1
id: "iss-2610121542258412"
slug: "the-issue-record-validator-internal-core"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "the build loop's stored-text sweep on fix/sec-build-next-series, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/issuerecord/validate.go"
remedy: "Quote every stored value the validator echoes with %q (or mask it with termsafe.Sanitize), with a test feeding U+202E and U+009B in schema_version and another field."
---

The issue record validator (internal/core/issuerecord) prints stored values unquoted in its error messages, schema_version among them, and those errors reach --json payloads and refusal reasons on several verbs, so a record carrying a control or bidi rune in a stored value puts it raw on those streams.

Security scope (abcd-fixes-what-it-adds): the route does not exist without abcd. The text is echoed by abcd's own --json surface, so abcd added the route and fixes it at the boundary the text enters by; a fix changes nothing about how the repository works.
