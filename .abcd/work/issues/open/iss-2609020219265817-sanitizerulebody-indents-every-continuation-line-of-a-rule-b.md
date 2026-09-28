---
schema_version: 1
id: "iss-2609020219265817"
slug: "sanitizerulebody-indents-every-continuation-line-of-a-rule-b"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/rules.go"
deferred_after: "v0.11.0"
deferral_reason: "ruling owed to the product thinker (run A lane drainS2, 2026-09-26): are rule bodies rendered code-safe for a markdown reader (a relative indent of four or more, or a fence, which neutralises every construct but flattens every legitimately structured multi-line rule wherever the block is rendered), escaped construct by construct (fence-aware backslashes on ATX headings, including the first line, setext underlines and HTML h1-h6 blocks, which put escapes into the raw text the model reads and is complete only by enumeration), or left to the host line-start contract as now? No mechanical close leaves rules reading as they do."
---

sanitizeRuleBody indents every continuation line of a rule body by two spaces, which defuses the line-start contract the host-side parser splits on but not CommonMark: a two-space-indented heading line inside a list item still renders as a heading INSIDE that item. A repo-overridden domain can therefore put an unmarked heading in front of any reader that renders the injected block as markdown, wearing no repo-override label of its own. The line-start contract holds and this is already stronger than the pre-fix behaviour, which indented nothing at all; closing the CommonMark half needs a decision on whether rule bodies get escaped, fenced, or left as the parser contract defines them. Evidence: sanitizeRuleBody in internal/core/rules/rules.go.

Addendum (autonomous run A, lane drainS2, 2026-09-26): the CommonMark half is wider than continuation lines. The FIRST line of a body is defused only for the line-start contract: in CommonMark `- # x` is a list item whose content is a heading, so a one-line rule forges a heading too. A setext underline (`===` or `---` under a text line) and an HTML block opening `<h2>` do the same from any line, and an HTML block of that kind can interrupt a paragraph. That is why the fix is a ruling rather than a mechanical close: closing every construct means either a code-safe rendering that flattens legitimate structure, or a fence-aware escaper that is complete only by enumeration and changes the raw text the model reads.
