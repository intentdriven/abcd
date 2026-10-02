---
schema_version: 1
id: "iss-2609302306281487"
slug: "v0-12-0-docs-currency-minors-deferred-past-the-tag"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.12.0 release gate docs-currency review (autonomous run A)"
origin: researcher-authored
production_mode: hand-written
found_at: "commands"
remedy: "Correct each page line against the binary as its finding states (argument-hints, flag lists, the JSON key, the project name, the User input line), and correct dc-3/dc-4/dc-24 in the next release's records rather than editing the shipped v0.12.0 section; docs-lint and the plugin page tests stay green."
resolution: "Ten page lines corrected against the binary (dc-6, dc-8, dc-10, dc-11, dc-12, dc-13, dc-14, dc-16, dc-21, dc-22). Three are not page fixes and are corrected here instead, since CHANGELOG.md and RELEASE.md are derived by the cut and the shipped v0.12.0 section is not rewritten: the word maintainer is refused by a banned-token pattern in this repository's own .abcd/docs-lint.json, not by a rule bundled with abcd's docs-lint (dc-3); the v0.12.0 preamble's claim that changes to earlier behaviour go unclaimed sits above two Breaking bullets filed under Added (dc-4); and after 6a40c898b only the UserPromptSubmit salvage hook declares the 120-second timeout, while PreToolUse and PreCompact keep the host default, not all three as iss-323's resolution says (dc-24)."
impact: fix
resolved_by:
  commit: "d0fa0fb95"
---

Thirteen v0.12.0 docs-currency findings are deferred past the tag, all minor or nitpick: dc-3 (CHANGELOG.md:37 and RELEASE.md:35 call a role-word ban a bundled docs-lint rule; it is a pattern in this repository's .abcd/docs-lint.json), dc-4 (CHANGELOG.md:21 preamble vs the two breaking bullets filed under Added), dc-6 (commands/ahoy.md:23-24 a status argument the binary refuses), dc-8 (commands/ahoy.md:4 argument-hint omits credential), dc-10 (commands/capture.md:4 argument-hint omits mentions), dc-11 (commands/capture.md:633-635 widening_runs sits under reading_outstanding), dc-12 (commands/docs.md:4,75-118 omit docs fidelity --intent), dc-13 (commands/history.md:4 ingest hint omits the required --into), dc-14 (commands/implement.md:42-54 omit join --model and --reason), dc-16 (commands/ingest.md:4,119 no User input line for its argument-hint), dc-21 (commands/launch.md:4 argument-hint omits receipts), dc-22 (commands/launch.md:1035 names the project abcd-cli), dc-24 (resolved iss-323's resolution says all three salvage entries declare the 120-second timeout; only UserPromptSubmit does since 6a40c898b). Found by the v0.12.0 release-gate docs-currency review (Fable 5.1, tier full) over b89784c4.

## Grounds

- pursued: each corrected page line now states what the binary does; a page hint naming a flag or sub-verb the binary refuses, or one omitting a required flag, would show it wrong
