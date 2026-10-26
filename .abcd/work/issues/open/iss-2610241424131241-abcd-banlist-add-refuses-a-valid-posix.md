---
schema_version: 1
id: "iss-2610241424131241"
slug: "abcd-banlist-add-refuses-a-valid-posix"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "hand-rolled private banlist prefill, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/banlist/engine.go"
remedy: "Skip an escaped parenthesis when screening for a (?...) group (scan for an unescaped '(' followed by '?'), with a test that \\(?x\\)? is accepted and (?i)x is still flagged."
---

abcd banlist add refuses a valid POSIX ERE that contains an escaped parenthesis followed by a question mark, such as \(?ar\)? for an optional literal parenthesis: checkPattern's static screen tests strings.Contains(pattern, "(?"), which matches the escaped form as if it were a Perl (?...) group, and the private layer refuses it as Perl-style.
