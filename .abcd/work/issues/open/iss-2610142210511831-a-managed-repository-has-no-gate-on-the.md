---
schema_version: 1
id: "iss-2610142210511831"
slug: "a-managed-repository-has-no-gate-on-the"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "v0.15.0 re-apply of the held outbound commit-msg gate"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/outbound.go"
related_issues: [iss-2609250834251447, iss-2608282124231752]
remedy: "Opt-in, on the --outbound-hook terms: wire scanner.ScrubOutbound (or abcd lint outbound) in front of the forge CLI's posting commands (pr create, issue create, pr comment, issue comment, and their edits), behind an ahoy install flag that is never a default, finding abcd only through git config --local abcd.binary and failing open with one warning line; prove it with fixture tests for a body carrying a session URL and a tool footer refused or stripped before posting, and a missing binary passing with the warning."
---

A managed repository has no gate on the text handed to the forge CLI for a pull-request body, an issue or a comment, so a live agent-session URL or a tool attribution footer reaches the forge unchecked, and the forge keeps the pre-edit revision of whatever was posted. The commit-message half of iss-2609250834251447 shipped as the opt-in ahoy install --outbound-hook gate; this is the forge-text half that record named and that gate does not cover. scanner.ScrubOutbound already defines the scrub for outward text, and abcd lint outbound the check, but nothing a managed repository runs puts either in front of the forge CLI's pull-request, issue or comment commands. Split out of iss-2609250834251447 when its commit-message half was resolved.
