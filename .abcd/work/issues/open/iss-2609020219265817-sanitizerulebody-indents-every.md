---
schema_version: 1
id: "iss-2609020219265817"
slug: "sanitizerulebody-indents-every"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/rules.go"
remedy: "Waits on ruling AN: per answer, in sanitizeRuleBody (internal/core/rules/rules.go): if code-safe, render each override rule body inside a tilde fence one character longer than the longest tilde or backtick run in the body, so no CommonMark construct survives; if escaped, backslash-escape a leading '#', a setext underline and a leading '<', accepting that CommonMark honours no escape inside raw HTML; if left as is, state the line-start contract as the boundary in the brief's configuration chapter. Prove the chosen form with a table test pinning the rendered bytes of a '- # x' first line, a setext underline and an '<h2>' block."
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (run A lane drainS2, 2026-09-26; rulings-owed AN): are rule bodies rendered code-safe for a markdown reader (a relative indent of four or more, or a fence, which neutralises every construct but flattens every legitimately structured multi-line rule wherever the block is rendered), escaped construct by construct (fence-aware backslashes on ATX headings, including the first line, setext underlines and HTML h1-h6 blocks, which put escapes into the raw text the model reads and is complete only by enumeration), or left to the host line-start contract as now? No mechanical close leaves rules reading as they do."
---

sanitizeRuleBody indents every continuation line of a rule body by two spaces, which defuses the line-start contract the host-side parser splits on but not CommonMark: a two-space-indented heading line inside a list item still renders as a heading INSIDE that item. A repo-overridden domain can therefore put an unmarked heading in front of any reader that renders the injected block as markdown, wearing no repo-override label of its own. The line-start contract holds and this is already stronger than the pre-fix behaviour, which indented nothing at all; closing the CommonMark half needs a decision on whether rule bodies get escaped, fenced, or left as the parser contract defines them. Evidence: sanitizeRuleBody in internal/core/rules/rules.go.

Addendum (autonomous run A, lane drainS2, 2026-09-26): the CommonMark half is wider than continuation lines. The FIRST line of a body is defused only for the line-start contract: in CommonMark `- # x` is a list item whose content is a heading, so a one-line rule forges a heading too. A setext underline (`===` or `---` under a text line) and an HTML block opening `<h2>` do the same from any line, and an HTML block of that kind can interrupt a paragraph. That is why the fix is a ruling rather than a mechanical close: closing every construct means either a code-safe rendering that flattens legitimate structure, or a fence-aware escaper that is complete only by enumeration and changes the raw text the model reads.

## Remedy grounds (2026-09-29)

- Why: the record's three forks, each stated with the change and the test that proves it; ruling AN (for the technical facilitator) is unanswered, so no fork is picked.
- Sources (consulted 2026-09-29): OWASP GenAI LLM01:2025, mitigation 6 'segregate external content' (https://genai.owasp.org/llmrisk/llm01-prompt-injection/); CommonMark 0.31.2, section 2.4 (backslash escapes do not work in raw HTML) and section 4.5 (a closing fence must be at least as long as the opening one) (https://spec.commonmark.org/0.31.2/). A computed-length fence is the only form the specification shows to be complete without enumeration; escaping is incomplete by the specification's own statement for HTML blocks.
- Rejected: a third-party Markdown sanitiser, a new dependency needing sign-off for a problem a fence closes in a few lines.
