---
schema_version: 1
id: "iss-2608301349335095"
slug: "an-indented-code-block-in-a-committed-issue-record-turns-the"
severity: "minor"
category: "process"
source: "impl-review"
found_during: "itd-189-round-3-remediation"
found_at: ".abcd/work/issues/open/iss-2608301308367566-record-stores-values-are-not-containment-checked-so-a-commit.md"
resolution: "The one record carrying the shape gets a fenced block, so site-render passes and preflight is green again. The wider gap it exposed — no gate reads a record body as markdown before the site render — is iss-2608301350287219 and stays open."
impact: internal
resolved_by:
  intent: "itd-189"
---

an indented code block in a committed issue record turns the site-render preflight gate red, and nothing on the capture path refuses one

`make preflight` exits 2 on `site-render` for every checkout of build/itd-189
since 070e1bd0, which committed a record whose body indents a two-line sample of
gate output by four spaces. The site renderer reads a fixed subset of markdown
and refuses an indented code block, naming the fence as the remedy.

The record is the only one of the 928 in the ledger that carries the shape, so
the immediate fix is one fence. The gap it exposes — that no gate reads a record
body as markdown before the site render — is iss-2608301350287219.

Found while remediating the itd-189 round-2 findings; the record it sits in is
unrelated to that work (it is the record-stores containment finding), and the
fence lands without touching what the record says.
