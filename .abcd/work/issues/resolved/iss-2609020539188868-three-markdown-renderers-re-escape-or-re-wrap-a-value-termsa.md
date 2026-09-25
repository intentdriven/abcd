---
schema_version: 1
id: "iss-2609020539188868"
slug: "three-markdown-renderers-re-escape-or-re-wrap-a-value-termsa"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/memory/schema.go"
resolution: "The memory renderers (RenderIndex, RenderContradictions, Ask's cited and empty renders) wrap cleaned names through termsafe.CodeSpan and clean every untrusted field with CleanProse. The lifeboat half is captured separately as iss-2609251355497247 for a lifeboat lane."
impact: fix
resolved_by:
  commit: "bf6df37b"
---

Three markdown renderers re-escape or re-wrap a value termsafe already cleaned, which is the class the intent audit's renderEvidence just had fixed. termsafe's guarantees hold over the exact string CleanProse returned; a caller that adds its own delimiters is parsing a different string than the cleaner reasoned about. memory.RenderIndex and memory.RenderContradictions (internal/core/memory/schema.go, the two backtick-wrapped format strings) wrap a cleaned page filename in their own backticks, so a filename carrying a backtick shifts code-span parity in the committed .abcd/memory/index.md and contradictions.md and can move a sheltered angle bracket out of its span; memory.Ask's match render has the same shape over a value cleaned only by Sanitize; lifeboat's synthesis_review severity bracket and press-release subhead emphasis are the weaker form (neither delimiter affects span parity, and the artefact is not a committed record), and synthesis_review renders f.ID through Sanitize alone, never CleanProse, so it can still carry an HTML comment opener or link syntax; lifeboat's synthesis_principles writes a cleaned principle as a bare paragraph with no leading-marker escape. Out of scope for the code-span fix that found them (a different package, a different caller, and none is a defect that change introduced). What the fix DID close for all of them is the same-line embedding half: no cleaned field can carry an unpaired backtick run any more, so two cleaned values on one line cannot re-pair. The fix must establish that no renderer alters a cleaned value's bytes, and that every untrusted field on a committed markdown line goes through CleanProse rather than Sanitize alone.

## Grounds

- pursued: a page name carrying a balanced backtick pair keeps its sheltered content inside one code span in index.md, contradictions.md and the ask answer; raw HTML or link syntax from a page field appearing live in any of the three would show it wrong
