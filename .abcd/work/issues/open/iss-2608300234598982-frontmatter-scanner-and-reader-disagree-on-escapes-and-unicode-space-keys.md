---
schema_version: 1
id: "iss-2608300234598982"
slug: "frontmatter-scanner-and-reader-disagree-on-escapes-and-unicode-space-keys"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "itd-182 third-round security review, 2026-08-30"
found_at: "internal/core/lint/schema.go (issueScalar), internal/core/frontmatter (keyRe), internal/core/capture (decodeScalar)"
---

Two pre-existing gate-versus-reader divergences reproduced on main, not introduced by the lapsed_at work: the record-lint issueScalar strips quotes with strings.Trim and never unescapes, so a double-quoted value carrying a backslash escape is lint-red while the reader decodes it (severity: "min\or" reproduces it); and a key line led by a Unicode space is ignored by frontmatter keyRe but read by capture after TrimSpace, so a Unicode-space-led bogus key is lint-green while the reader refuses and skips the record. The first shares its root with iss-2608300205044566 and should fold into it; the second applies to every key.
