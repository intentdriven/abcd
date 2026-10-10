---
schema_version: 1
id: "iss-2610101930219027"
slug: "an-atx-heading-line-inside-an-html-block-ends-the"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/project.go"
remedy: "Refuse, in verifyRedaction beside fenceInHTMLBlock, an ATX heading line that sits inside an HTML block of the original document (the same htmlBlockEnd walk), so the redactor's and a renderer's disagreement about where a section ends fails closed; a test assembles the reported input and expects a refusal naming the line."
resolution: "verifyRedaction now refuses an ATX heading line inside an HTML block of the original document (headingInHTMLBlock over the htmlBlockInterior walk it shares with fenceInHTMLBlock), so the redactor's and a renderer's disagreement about where an excluded section ends fails closed; TestAHeadingInsideAnHTMLBlockRefuses assembles the reported input."
impact: fix
---

An ATX heading line inside an HTML block ends the cold-reading redactor's span early, so the rest of an excluded section travels. The redactor spans sections by the site section walk (sectionSpan in internal/core/reading/project.go over site.Sections), which reads a column-0 '## Next' as a heading even inside a `<div>` block, while a renderer reads that line as raw HTML. For `## audit notes / SECRET / <div> / ## Next / SECRET-IN-DIV / </div> / ## After / KEPT`, SECRET-IN-DIV travels. The verifier refuses a fence opener inside an HTML block (fenceInHTMLBlock) but has no heading-in-HTML-block refusal, and the redacted text it reads has already lost the `<div>` line. Pre-existing on main; the case-insensitive redaction made it reachable for case-variant headings too.
