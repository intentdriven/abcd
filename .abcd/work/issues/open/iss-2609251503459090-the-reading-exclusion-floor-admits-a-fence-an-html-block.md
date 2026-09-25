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
---

The reading exclusion floor admits a fence an HTML block swallows, and the excluded section travels. A fence opener written directly under a line that opens a CommonMark HTML block (types 6 and 7, e.g. <div> or <span>, and the <!, <? and </ starts) is raw HTML to a renderer, whose block runs to the first blank line, so a heading after that blank line is live; every mdrecord reading sees a fence and masks it, so the redactor (site.Sections) and the verifier (floorFences) agree it is an example and redactExcluded returns nil with the private section in the bundle. Probe: '<div>' then a backtick fence holding a blank line and '## Private Notes'; the '<span>' plus tilde form regressed on fix/one-fence-rule (the old toggle never saw tildes). The lane's ruling is that any ambiguity fails closed: a fence opener inside an HTML block's run of non-blank lines is refused, naming the line.
