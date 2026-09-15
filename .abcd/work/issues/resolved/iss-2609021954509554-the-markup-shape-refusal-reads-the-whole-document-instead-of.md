---
schema_version: 1
id: "iss-2609021954509554"
slug: "the-markup-shape-refusal-reads-the-whole-document-instead-of"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "cold-reading Phase A, pre-PR review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/project.go"
resolution: "verifyRedaction now runs the markup-shape scan over the unfenced body: a new unfencedBody helper blanks every fenced line rather than dropping it, so the fenced example says nothing to the scan while a match offset still counts the document's own newlines. TestAFencedMarkupExampleIsNotTheShape in project_test.go holds both halves — a record carrying a fenced <img alt= example assembles, and the same shape outside a fence still refuses naming it. The dead fenceMask function, whose only caller bodyFenceMask replaced, is deleted with it, and the two comments that named it now describe the mask rather than the symbol."
impact: fix
---

The markup-shape refusal reads the whole document instead of the unfenced body, so a fenced HTML example refuses every assembly. verifyRedaction in internal/core/reading/project.go calls markupShapeOf over strings.Join(lines) and ignores the fence mask it already computed, unlike rawHTMLHeading and the indented-ATX and excluded-key scans beside it, which all take the mask. A code block showing an HTML attribute whose value opens on the line after its equals sign is an EXAMPLE, not markup the bundle carries; any admitted markdown document holding one refuses the run at every position. Expected: the shape scan reads the unfenced body, with fenced lines blanked rather than dropped so a line number still names the document's own line, and the same shape outside a fence still refuses.

## Grounds

- pursued: a fenced code block showing markup is an example and never material the mask must bound, so an admitted markdown document holding one assembles at every position while unfenced markup of the same shape still refuses. What would show it wrong: an assembly refused over a document whose only occurrence of the shape is inside a fence, or an unfenced occurrence admitted.
