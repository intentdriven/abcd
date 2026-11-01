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
resolution: "The link-targets gate now walks every rendered page for its /docs/ links, the record pages included, while the composed-text gates still walk only the composed pages; an unparsed record page is denied the link-targets pass, and the site brief, internals chapter and lint command page state the new scope."
impact: fix
resolved_by:
  commit: "d1787c841af9f80ec4fc43bd17fbd953af8647c1"
---

The site lint's link-targets gate judges only the composed pages, so a /docs/ link the verbatim record rendering writes through the same siteHref (the glossary's loop entry links /docs/explanation/process/ from internal/core/site/explorer.go) is never checked: renaming that docs page breaks the record link with every gate passing.

## Grounds

- pursued: a /docs/ link on a record page that names no built file fails link-targets on that record page (TestLinkTargetsJudgeTheDocsLinksOnARecordPage, failing before and passing after), and on the real tree with docs/explanation/process/index.html removed the glossary's loop page now fails where the base binary passed; it would be shown wrong by a record page's broken /docs/ link passing lint site once the docs tree is in the output, or by a composed-text gate raising a finding on a record page
