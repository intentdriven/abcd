---
schema_version: 1
id: "iss-2609261039139464"
slug: "a-principle-statement-s-citation-check-reads-links-as-inline"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-principles"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/project.go"
resolution: "Both readers widen to every link shape: the lint refuses inline, reference, autolink and bare-URL links in the statement; the projection unwraps labelled links and verifyPrincipleItem refuses one with no label."
impact: fix
resolved_by:
  commit: "a5f5a266662ed258f549a49a4847a391e68fd9f6"
---

A principle statement's citation check reads links as inline [label](target) only: a bare URL, an autolink <https://...> and a reference-style [label][ref] travel into the bundle raw and the lint is silent on all three, while the manifest's exclusion row asserts record handles and links in a principle stay behind.

## Grounds

- pursued: TestPrincipleStatementMayNotCite covers bare, www, autolink, email, reference, collapsed and nested-bracket links; TestReferenceLinksUnwrapInTheStatement and TestUnlabelledLinkInTheStatementRefuses hold LEAKREFLABEL, LEAKURL and LEAKAUTO; a URL or a ][ reaching a bundle would show it wrong
