---
schema_version: 1
id: "iss-2611060809502312"
slug: "the-drive-section-of-35-drain-md-lines"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/35-drain.md"
remedy: "Add the drive's default cap, consent, exit statuses, remaining refusals and `--until` forms to the drive section of the chapter."
---

The drive section of `35-drain.md` (lines 193-340) states the walls, a credential variable and a start from an agent's session as refusals, and its appendix leaves exit codes to the prose. It never states the default cap of eight lanes (`driveDefaultCap`, against no cap on the bare verb); `--yes` and the refusal with no terminal to ask at; exit 3 for a second drive and for a drive stopped, stopped short or holding an issue back, exit 2 for a failed check, exit 130 for the second signal; the refusals for an agent runner not enabled or not admitted and for a role pinned to the host; or the `--until` forms (a date read as the end of that day, or an RFC 3339 instant) and the refusal of a time already past. `abcd drain --help` and `commands/drain.md` state each.
