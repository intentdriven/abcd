---
schema_version: 1
id: "iss-2610140706347640"
slug: "a-garbled-release-mark-drops-the-floor"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "cut 2 gate review of the installer round, 2026-10-14"
origin: researcher-authored
production_mode: hand-written
found_at: "the ahoy release-mark reader (ReadReleaseMark in internal/core/ahoy, landing with the cut-2 installer fixes)"
remedy: "When the release mark exists but cannot be read (unparsable tag, missing tag line), keep the fail-open but say so: name the mark file and that the rollback floor is not in force, in ahoy install's summary and in update's output, the way an unreadable owned-copy declaration is reported."
---

A garbled release mark makes the rollback floor disappear without a word. ReadReleaseMark in internal/core/ahoy reads an unparsable mark as no mark, so ahoy install and update adopt any genuine release, however old, and nothing tells the person their newest-release floor is gone. The fail-open is deliberate and safe: the mark is read only when this account owns it and no one else can write it, so whoever can garble it could already delete or lower it, and a copy admitted this way still passes the provenance check for the release it names (the warm path and hooks/bootstrap.sh fail open the same way). What is missing is the diagnostic. Route without abcd: none; the mark and its floor are abcd's own, so abcd fixes it.
