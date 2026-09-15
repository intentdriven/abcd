---
schema_version: 1
id: "iss-2608301450065320"
slug: "the-assembler-s-key-and-heading-signals-are-scoped-to-markdo"
severity: "major"
category: "security"
source: "user-observation"
found_during: "itd-183-fidelity-audit"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/project.go"
resolution: "The scope decision is declared rather than inferred. Every include row carries a Scan declaration saying whether the exclusion floor parses what the row admits, collect runs redactExcluded over exactly the rows declared parsed, and redactExcluded's own file-extension test is gone — so admission and examination are one declaration instead of a row on one side and an extension test on the other. An item from an unparsed row travels whole and its manifest entry is marked unscanned, and the manifest's key and heading exclusions are asserted for the items marked parsed and for no other. The Go fixture at itm-0736 therefore arrives disclosed rather than silently, which is the ruling of 2026-09-02."
impact: fix
resolved_by:
  intent: "itd-194"
  spec: "spc-2609021003136831"
---

the assembler's key and heading signals are scoped to markdown so a record-shaped document inside a Go fixture carries its Audit Notes into the bundle while the manifest asserts refusal

Found by the itd-183 fidelity audit (rcp-139d9b40f0e8), which ran the
instrument rather than reading it.

**This is the tenth floor record and the only one that is LIVE ON THE CORPUS.**
The other nine are recognition gaps on spellings no committed record writes.
This one fires today, on this repository, at HEAD.

The key and heading signals are scoped to markdown (project.go:57). So
`internal/core/site/fixture_test.go` — a Go file holding record-shaped fixture
text — carries three literal `## Audit Notes` sections, WITH THEIR ACCEPTANCE
ROLLUPS, into the bundle at item `itm-0736`, while the same run's manifest
asserts that `Audit Notes` was refused. Manifest and bundle disagree, and the
manifest is the side that is wrong.

Mitigating, and why this is major rather than critical: the content is
synthetic fixture text, not a real audit; the bound is disclosed at
`.abcd/development/readings/README.md:73`; and no reading runs this cycle, so
nothing consumes the bundle. The auditor was explicit that "no reading runs
this cycle" bears on the RISK, not on the verdict — the trigger is "when the
assembler runs", and it ran.

Why it belongs to the floor intent rather than to a fix here: the class is the
same one the other nine share — the floor recognises a construct by the shape
of the file it sits in rather than by what the file CONTAINS. A `.go` file can
hold a record; so can a `.json` fixture, a testdata blob, or a doc comment.
Narrowing to markdown is a scope decision that the include table makes and the
floor inherits without stating.

Remedy is a design question, not a patch: either the include table admits only
what the floor can read, or the floor reads what the include table admits.
Seed material for the exclusion-floor intent, and the strongest single argument
in it — because it is the one leak a maintainer can reproduce today.

## Grounds

- pursued: we expect moving the scope decision onto the table row, and marking every unexamined item in the manifest, to close the container-shape class where widening the parse would have been the other branch, because the manifest then states a fact about each item rather than a claim about the run; an item the manifest marks parsed that no scan ran over, or an unparsed item arriving with no mark, would show it wrong
