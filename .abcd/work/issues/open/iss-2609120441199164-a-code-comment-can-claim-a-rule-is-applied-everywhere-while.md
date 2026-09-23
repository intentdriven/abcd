---
schema_version: 1
id: "iss-2609120441199164"
slug: "a-code-comment-can-claim-a-rule-is-applied-everywhere-while"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "second autonomous-run field experiment in a managed repository, 2026-09-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/repolint"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (planning owed: a detector listing universality claims in comments near sibling constructs). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

A code comment can claim a rule is applied everywhere while a sibling file in the same package breaks it, and nothing detects the gap. Found by a reviewer reading the comment against the code, in a managed repository, not by any scan. abcd already holds the idea this needs: the citation merge carries honesty rules, and the documentation gate refuses a change-narration claim in prose. Neither reaches a scope claim written in a Go comment, so the most load-bearing sentences in the codebase, the ones that tell the next reader an invariant holds across every call site, are exactly the ones nothing checks. The class is worth naming because this repository keeps meeting it from both ends. Its own bug-hunting playbook states the rule, that a fix is not done until every sibling site is swept and the pattern is grepped rather than the instance, and a defect on this very branch was precisely that shape: the anti-forgery work fenced a renderer's block content, left its block metadata raw, and the regression test covered only the path that was fixed, so a comment and a test together asserted a guarantee the code did not hold. A detector cannot judge whether an invariant is true, but it can find the claim: a comment asserting universality near a construct that has siblings is a place for a human to look, and listing those places is cheap where reading every comment is not.
