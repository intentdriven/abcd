---
schema_version: 1
id: "iss-2610151050395768"
slug: "the-surfaces-register-is-the-last-shared"
severity: "minor"
category: "tech-debt"
source: "agent-observation"
found_during: "v0.15.0 run 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md"
remedy: "Make the surfaces register's chapter table and its two inline verb lists one entry per line in a stable order (by chapter name, as the release-gate manifest now is), or generate them from the chapter files, so two branches that each add a surface edit different lines."
---

After the release-gate manifest's derived values stopped being stored (iss-2610150526192138), the brief's 04-surfaces/README.md register is the main remaining line every surface-adding branch edits: its chapter table ends on the newest number and two of its rows are single-line inline lists of verbs, so two such branches conflict there. Measured on the ten surface-adding pull requests of 2026-09-24 to 2026-10-15: the table end in 8, the agents-and-hosts row in 5, the one-file-per-verb list in 5.
