---
schema_version: 1
id: "iss-2610141947553722"
slug: "abcd-lint-s-privacy-hygiene-rule-reads-a"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "hosted-sites build, bare abcd lint run"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/research/notes/2026-10-14-abcdev-app-github-builds-and-cache.md"
remedy: "Treat a dotted number as an address only where its context reads as one (not after 'section', '§' or 'RFC N'), or skip prose files' section citations, with a test on '[redacted-address]'"
refines: [iss-2609290845269642]
---

abcd lint's privacy-hygiene rule reads a dotted RFC section number such as [redacted-address] in prose as a non-reserved IPv4 address and fails the file
