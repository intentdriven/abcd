---
schema_version: 1
id: "iss-2609011026401952"
slug: "the-implement-loop-has-no-step-that-ships-an-intent-once-its"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "The implement loop now states its last step everywhere the loop is stated: commands/intent.md gains a Ship section that runs abcd spec close <spc-N> in the same change that lands the work and says why (launch ship composes from terminal folders only, so a planned intent with its code on main ships with no changelog line and the cut exits 0); AGENTS.md's definition of done carries the rule beside the issue-resolution rule it mirrors; the brief's lifecycle diagram marks the close as a manual step nothing runs for you; and the bundled INTENTS rule domain injects the reminder on intent-related prompts. No new verb was needed: spec close already ships the linked intent as its close-hook."
impact: fix
---

The implement loop has no step that ships an intent once its work merges: abcd intent plan moves drafts to planned and abcd spec close moves planned to shipped, but nothing in the build-review-merge routine invokes the second, so an intent whose code is on main sits in planned indefinitely and launch ship composes the changelog only from terminal folders. Two intents delivering a BREAKING CLI change were about to be released with no changelog line, and the omission is invisible because the cut exits 0 - a planned intent is not a refusal, it is simply not seen

## Grounds

- pursued: the verb exists and the omission is procedural, so documenting the step where the loop is documented closes the gap without a design decision; a mechanical gate that refuses a merged intent whose spec is still open would be the durable fix and is left open as the question underneath
