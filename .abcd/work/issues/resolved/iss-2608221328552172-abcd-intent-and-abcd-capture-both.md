---
schema_version: 1
id: "iss-2608221328552172"
slug: "abcd-intent-and-abcd-capture-both"
severity: "major"
category: "observation"
source: "user-observation"
found_during: "adversarial-review-of-2026-08-22-filings"
found_at: "internal/surface/cli/cli.go"
resolution: "Both verbs now refuse a lone bare token — one whitespace-free positional — at exit 2 with nothing written, and the tree-wide sweep that asserts every parent refuses an unknown sub-verb no longer exempts capture and intent, so the detector is not verb-specific."
impact: fix
resolved_by:
  commit: "ae727d8"
---

abcd intent and abcd capture both swallow an unrecognised sub-verb as content and mint a record at exit 0, violating unrecognised-input-never-writes on the two highest-traffic mutating verbs. Verified on 45afe46 against a freshly built binary: 'abcd intent nosuchthing' prints 'created itd-148 (drafts)' and exits 0; 'abcd capture nosuchthing' prints 'captured iss-... (open)' and exits 0. Both records were durable files on disk, removed by hand afterwards. The principle at development/principles/unrecognized-input-never-writes.md states the rule -- a verb that mutates state fails closed, anything not exactly recognised is an error, never a fallback interpretation -- and cites this precise scenario as its founding evidence: a misspelled capture subcommand swallowed as capture text, filing an issue when the user asked to resolve one. The principle is therefore documented and unenforced, and the failure it was written about is still live on both verbs. The cause is structural rather than accidental: both verbs take free text as their canonical create path, so an unknown token is indistinguishable from intended content. That makes a bare token ambiguous by construction, which is the design question a fix must answer -- candidates include requiring quoting or an explicit create flag for the text path, refusing a single bare token that matches no sub-verb and resembles one, or accepting text only from stdin. A typo costs a spurious durable record, a burnt id under the max+1 intent allocator, and a record-lint index_drift blocker until the stray is noticed. Related: adr-45 on id allocation.