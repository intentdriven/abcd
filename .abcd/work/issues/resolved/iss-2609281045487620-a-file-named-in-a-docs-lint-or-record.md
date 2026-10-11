---
schema_version: 1
id: "iss-2609281045487620"
slug: "a-file-named-in-a-docs-lint-or-record"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25: lane roles"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/lint.go"
resolution: "a non-markdown file in roots is a configuration error pointing at extra_roots, in LintAt and DocumentsInRoots"
impact: fix
resolved_by:
  commit: "79923a98a309874d0cfb33de857dba8c818c2c7a"
---

A file named in a docs-lint or record-lint config's roots that is not markdown is read as nothing and reported clean: the per-root walk keeps only .md files, so a root such as .abcd/rules.json passes the does-not-exist check, contributes zero documents, and every rule the config arms reads none of it while the lint exits 0. Confirmed by a test on a scratch copy (roots [docs, rules.json] with a ban matching the JSON: no finding, no error; DocumentsInRoots counts it as zero). Found while widening the role ban past the documentation, where the spec's 'roots widened to .abcd/rules.json' would have silently checked nothing. Wanted: a non-markdown file in roots is a configuration error that points at a token's extra_roots, the way a missing root already is (GitHub #360).

## Grounds

- pursued: we expect a config naming a non-markdown file in roots to fail loud rather than report clean; shown wrong if TestRootsRefuseANonMarkdownFile passes while such a root lints clean
