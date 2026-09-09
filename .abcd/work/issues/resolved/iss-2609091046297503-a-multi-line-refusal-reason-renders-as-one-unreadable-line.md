---
schema_version: 1
id: "iss-2609091046297503"
slug: "a-multi-line-refusal-reason-renders-as-one-unreadable-line"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "adversarial-review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/release"
resolution: "renderCut routes a multi-line refusal reason through termsafe.SanitizeBlock, which neutralises a control sequence while leaving the line structure intact, so the surface guard's break list, the stale-intent refusal and the findings gate's record enumeration each render as the lines they were written as."
impact: fix
resolved_by:
  commit: "c28300c565a4"
---

renderCut passed a refusal reason through the terminal sanitiser before splitting it on newlines, and the sanitiser masks a newline to a question mark, so the split found nothing to split and every multi-line reason printed as a single run-on line with question marks where its line breaks were. The reasons affected are the ones a reader most needs to act on: the surface guard's list of what narrowed since the anchor tag, the stale-intent refusal, and the findings gate's enumeration of the records blocking a cut, each of which is written as a heading plus one indented line per record and each of which arrived as one unreadable paragraph. Nothing was lost from the payload, so the JSON envelope carried the reason correctly all along and only the terminal render was affected, which is why it survived unnoticed: the machine-readable surface was right and the human one was not. The sanitiser has a sibling built for exactly this, SanitizeBlock, which neutralises a control sequence while leaving the line structure intact, so the fix is to route a multi-line reason through it and to keep the single-line sanitiser for values that are genuinely one line. Detector: a refusal whose reason carries newlines must render as that many lines with no question-mark substitution, and a single-line reason must render unchanged. Found while adding the release findings gate, whose own six-line refusal was the render that made it visible. The detector this record names is `TestRenderCutKeepsAMultiLineRefusalsLineStructure` and `TestRenderCutLeavesASingleLineRefusalUnchanged` in `internal/surface/cli/rendercut_refusal_test.go`, written on 2026-09-09 — the fix landed without one, so the record described a test that did not exist. A third, `TestRenderCutStillNeutralisesAControlSequenceInARefusal`, holds what the single-line sanitiser was there for: keeping the line structure must not also let an escape sequence through.

## Grounds

- pursued: we expect a refusal a reader must act on to render as the lines it was written as, so the records or surfaces it names can be read one per line rather than run together; a reason that still collapses, or a single-line reason that gains a spurious break, would show it wrong.
