---
id: spc-2609230612443069
slug: routine-prompts-carry-the-outbound-policy
intent: itd-152
origin: researcher-authored
production_mode: hand-written
---
# Routine prompts carry the outbound policy

## Summary

The remainder of [itd-152](../../intents/planned/itd-152-autonomous-cloud-runs-in-sibling-repos-leaked-harness-attrib.md)
that [spc-45](../closed/spc-45-autonomous-cloud-runs-in-sibling-repos-leaked-harness-attrib.md)
did not deliver. spc-45 closed on 2026-09-23 with acceptance criteria 1 to 4
delivered: the `scanner.ScrubOutbound` primitive, its session-URL and
clean-pass tests, and the shared harness-leak pattern set enforced by
`abcd lint`'s privacy rule and the record and docs lint `harness_leak` rule.
This spec carries the one criterion that did not ship.

## Scope

- **Acceptance criterion 5 (missing).** Given an autonomous routine prompt
  assembled for a managed repo, when the prompt is composed, then it carries
  the policy that bans session URLs and harness footers in public text and
  mandates a post-create re-read-and-strip of every PR, issue and comment
  the loop creates. At closure the policy exists as the `scanner.OutboundPolicy`
  constant (`internal/adapter/scanner/outbound.go:28`, pinned by
  `internal/adapter/scanner/harnessleak_test.go:228`), but no routine-prompt
  composer embeds it; only lint findings and reports quote it.
