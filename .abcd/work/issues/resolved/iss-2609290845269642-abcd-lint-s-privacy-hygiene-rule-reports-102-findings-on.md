---
schema_version: 1
id: "iss-2609290845269642"
slug: "abcd-lint-s-privacy-hygiene-rule-reports-102-findings-on"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/repolint/rule_privacy.go"
resolution: "The one real value (a home path of the authoring machine in two fold tests) moved to a persona home; Homebrew's Linux prefix reads as a system account in the scanner and the lint; every other finding is a persona placeholder or carries the rule's per-line waiver. abcd lint reports 0 errors and 0 warnings. Git history still holds the real value; a rewrite is the person's decision. The missing gate stays with iss-2608231000561060."
impact: fix
resolved_by:
  commit: "10d038127"
---

abcd lint's privacy-hygiene rule reports 102 findings on main (88 errors, 14 warnings) in this public repository's committed tree, and nothing gates it. Classified by the privacyLint lane of autonomous run A: 11 are class A, a real value from a real machine: two test files (the ahoy receipt-fold test and the fsutil redact-root fold test) build their fixture paths under the authoring machine's actual home directory, whose account name is a generic one the scanner itself lists as generic. 71 are class B, synthetic values whose shape carries the meaning and that carry no waiver: redactor and detector fixtures that need a non-persona, non-reserved value (a token unique enough that an absence assertion means something, a prefix collision, a traversal escape, a non-reserved address), and records, comments and research observations that illustrate such a case. 3 are class C, synthetic placeholders in comments and one test whose shape carries nothing, so a persona home serves equally. 17 are class D, not personal identifiers at all: public cloud metadata endpoints the URL guard must name, the Homebrew-on-Linux system prefix, the CI runner's system home, RFC section numbers read as IPv4, a hex fragment read as IPv6, and Go selectors read as LAN hostnames. Git history keeps every class-A value; rewriting published history is a decision for the person, not a lane. The missing gate is iss-2608231000561060.

## Grounds

- pursued: bare abcd lint exits 0 with no privacy-hygiene finding and every touched redactor test passes unchanged in value; a privacy-hygiene finding on this tree, or the authoring home path anywhere in the tree, would show it wrong
