---
schema_version: 1
id: "iss-2609251503459090"
slug: "the-reading-exclusion-floor-admits-a-fence-an-html-block"
severity: "major"
category: "security"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/project.go"
resolution: "The floor refuses a fence opener that sits inside an HTML block, naming its line: fenceInHTMLBlock walks the body tracking where a CommonMark HTML block ends (a blank line for types 6 and 7, the terminator for types 1-5) and refuses when an opener under either exported mdrecord rule falls inside one. The fences stay mdrecord's. A block that ends on its own line (a one-line comment, a closed declaration) is still admitted."
impact: fix
resolved_by:
  commit: "b4bc4cb2"
---

The reading exclusion floor admits a fence an HTML block swallows, and the excluded section travels. A fence opener written directly under a line that opens a CommonMark HTML block (types 6 and 7, e.g. <div> or <span>, and the <!, <? and </ starts) is raw HTML to a renderer, whose block runs to the first blank line, so a heading after that blank line is live; every mdrecord reading sees a fence and masks it, so the redactor (site.Sections) and the verifier (floorFences) agree it is an example and redactExcluded returns nil with the private section in the bundle. Probe: '<div>' then a backtick fence holding a blank line and '## Private Notes'; the '<span>' plus tilde form regressed on fix/one-fence-rule (the old toggle never saw tildes). The lane's ruling is that any ambiguity fails closed: a fence opener inside an HTML block's run of non-blank lines is refused, naming the line.

## Grounds

- pursued: the <div>+backtick, <span>+tilde, continued-block, open-declaration, processing-instruction and closing-tag shapes are refused naming the opener, while a fence after a blank line, after a one-line comment or after a closed declaration is admitted (TestVerifyRedactionRefusesAFenceAnHTMLBlockSwallows, TestVerifyRedactionAdmitsAFenceAfterAClosedHTMLBlock), and the cold-reading eval lane passes over the committed corpus; a swallowed opener admitted, or a committed corpus file refused, would show it wrong
