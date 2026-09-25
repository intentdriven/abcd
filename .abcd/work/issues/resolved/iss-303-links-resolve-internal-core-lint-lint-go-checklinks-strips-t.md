---
schema_version: 1
id: "iss-303"
slug: "links-resolve-internal-core-lint-lint-go-checklinks-strips-t"
severity: "nitpick"
category: "process"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: "internal/core/lint/lint.go"
resolution: "The link_anchors rule checks each link's fragment against the target page's heading slugs and HTML anchors, armed at warn in both lint configs."
impact: additive
resolved_by:
  commit: "57e3f44d"
---

links_resolve (internal/core/lint/lint.go checkLinks) strips the #fragment before resolving and skips same-file # links, so no gate validates heading anchors even though both record-lint and docs-lint declare the rule blocking and the Makefile advertises it as catching a broken relative link; ~18 broken anchors sit on a green tree. Proposed: extend the rule to slug the target file's ATX headings and validate the fragment, landing warn-first
## Evidence

- `internal/core/lint/lint.go` `checkLinks` strips the `#…?` fragment before `os.Stat` and
  `continue`s on same-file `#` links, so a heading anchor is never validated. Both
  `.abcd/record-lint.json` and `.abcd/docs-lint.json` mark `links_resolve` blocking, and the
  Makefile advertises docs-lint as catching "a broken relative link".
- Consequence: ~18 broken heading anchors sit in `.abcd/development` on a green tree (see
  iss-298), and the resolved iss-14 deep-link fix re-broke because nothing gates the class.

## Adversarial review

CONFIRMED as a feature gap (nitpick, ledger capture — not a code defect: the rule matches its
file-level contract) by an independent refuter. Proposed: an `anchor`-validating extension to
`links_resolve` that GitHub-slugs the target's fenced ATX headings, landed warn-first given
the existing residue. A candidate acceptance-corpus entry for iss-46.

## Grounds

- pursued: a fragment naming no heading of its target (or of the linking page) is reported at warn while a correct slug, a duplicate's -1 and an HTML anchor pass; a broken anchor on a green tree would show it wrong
