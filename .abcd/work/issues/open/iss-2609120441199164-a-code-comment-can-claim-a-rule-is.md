---
schema_version: 1
id: "iss-2609120441199164"
slug: "a-code-comment-can-claim-a-rule-is"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "second autonomous-run field experiment in a managed repository, 2026-09-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/repolint"
deferred_after: "v0.11.1"
deferral_reason: "planning F owed to the product thinker: no detector lists universality claims in code comments (repolint has none). It is a new detector with no planned intent. (re-checked at e792a2314 by lane drainDQ3, run A, 2026-09-29)"
remedy: "Waits on planning F (a new detector): if planned, add a report-only repolint listing of Go comments that assert universality (every, all, always, never, each call site or path) in a package where the named construct has sibling definitions, as places for a person to look rather than failures, proven by a fixture package with one claim beside a sibling (listed) and one without (not listed); if declined, wontfix naming the sibling-sweep step of the bug-hunting playbook as the control."
---

A code comment can claim a rule is applied everywhere while a sibling file in the same package breaks it, and nothing detects the gap. Found by a reviewer reading the comment against the code, in a managed repository, not by any scan. abcd already holds the idea this needs: the citation merge carries honesty rules, and the documentation gate refuses a change-narration claim in prose. Neither reaches a scope claim written in a Go comment, so the most load-bearing sentences in the codebase, the ones that tell the next reader an invariant holds across every call site, are exactly the ones nothing checks. The class is worth naming because this repository keeps meeting it from both ends. Its own bug-hunting playbook states the rule, that a fix is not done until every sibling site is swept and the pattern is grepped rather than the instance, and a defect on this very branch was precisely that shape: the anti-forgery work fenced a renderer's block content, left its block metadata raw, and the regression test covered only the path that was fixed, so a comment and a test together asserted a guarantee the code did not hold. A detector cannot judge whether an invariant is true, but it can find the claim: a comment asserting universality near a construct that has siblings is a place for a human to look, and listing those places is cheap where reading every comment is not.

## Remedy grounds (2026-09-29)

Why: a detector cannot judge whether an invariant holds but can cheaply find where one is claimed, which is the record's own framing, so the output is a listing, not a gate. Rejected: a warn rule under the BT3 baseline, whose new-warning-fails behaviour would turn a heuristic listing into a refusing gate.
