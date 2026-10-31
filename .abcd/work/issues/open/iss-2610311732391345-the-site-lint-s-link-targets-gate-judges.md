---
schema_version: 1
id: "iss-2610311732391345"
slug: "the-site-lint-s-link-targets-gate-judges"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "drain lane review of iss-2610252224215664 (PR #1297)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/linktargets.go"
remedy: "Widen the link-targets walk to the /docs/ links on every rendered page, record pages included, while keeping the record rendering exempt from the composed-text gates."
---

The site lint's link-targets gate judges only the composed pages, so a /docs/ link the verbatim record rendering writes through the same siteHref (the glossary's loop entry links /docs/explanation/process/ from internal/core/site/explorer.go) is never checked: renaming that docs page breaks the record link with every gate passing.
