---
schema_version: 1
id: "iss-2609250850380420"
slug: "this-repository-s-own-committed-githooks-pre-commit-name"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".githooks/pre-commit"
resolution: "The four hardening fixes the template received on 2026-08-26 are ported into this repository's .githooks/pre-commit (function pin and sweep, gitlink path scanned before the skip, control-byte scrub) and the pin into .githooks/pre-merge-commit, with a test driving all four against this repository's hook."
impact: fix
resolved_by:
  commit: "1fea1d61228f095175d1af26bf1756c8fdda6787"
---

This repository's own committed .githooks/pre-commit name guard lacks the four hardening fixes the scaffolded template received on 2026-08-26 (f95650de): it does not pin read, echo, exit, test and [ against inherited shell functions or sweep the survivors, so a BASH_ENV defining read and echo makes it read zero entries and commit a banned name, and a shadowed declare plus exit turns a printed refusal into a commit; it appends a staged path after the gitlink skip, so a submodule path carrying a banned name is never scanned; and it echoes staged paths raw, so control bytes in a refused path can forge the refusal text. The fix landed on the template alone, and its test drives only the template, so nothing held the dogfood copy to it. The .githooks/pre-merge-commit half lacks the same pin.

## Grounds

- pursued: this repository's guard now refuses under a read/echo or declare/exit shadow, scans a gitlink path and never echoes a control byte; any of the four BASH_ENV cases passing a banned name or a raw ESC would show it wrong
