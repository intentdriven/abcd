---
schema_version: 1
id: "iss-2610090748299348"
slug: "a-public-source-behind-an-idea-is-named-in-the-repository"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "2026-10-09 product thinker capture"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/ideate.md"
remedy: "Tie source naming to adoption: a killed or abandoned verdict stays in the home store with its sources and never moves to the repository; publishing a verdict and the ACKNOWLEDGEMENTS.md entry happen only for an accepted idea (iss-2610090742488191), and the publish step refuses a verdict whose outcome is killed."
---

A public source behind an idea is named in the repository only once the idea is accepted. An idea that is rejected, killed or abandoned names none of its sources anywhere public (no record, research note, decision line, commit or pull request), only in its private verdict; nothing enforces this today, and ideate record currently writes every verdict's sources into the committed research notes.
