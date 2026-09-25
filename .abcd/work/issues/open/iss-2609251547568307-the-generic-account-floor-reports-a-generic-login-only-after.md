---
schema_version: 1
id: "iss-2609251547568307"
slug: "the-generic-account-floor-reports-a-generic-login-only-after"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/identity.go"
---

The generic-account floor reports a generic login only after a home root, a tilde or in an address's local part, so the account positions a shell session or a config dump writes pass unreported: a home root with no leading slash (an archive listing's Users/<login>/Desktop, a relative home/<login>/.config), a key/value position (USER=<login>, LOGNAME=<login>, username: <login>, login: <login>, a JSON "user": "<login>"), and the argument of an account command (su - <login>, chown <login>:staff, chown -R <login>). Each is where a login stands, not a word in prose, and a caller whose login is on the generic list leaks it from every one of them through capture and history. Found by the scanner-cluster review (floor gaps, item 3).
