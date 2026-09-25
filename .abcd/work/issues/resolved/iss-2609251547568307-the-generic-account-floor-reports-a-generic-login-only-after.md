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
resolution: "standsAsAccountName accepts three more account positions for a generic login: a bare home root that begins a token (afterBareHomeRoot: Users/LOGIN/..., home/LOGIN/...), the value of a key naming the user (afterAccountKey: user, username, login, logname, quoted or not, with ':' or '='), and the first operand of su or chown past up to three options (afterAccountCommand). Every walk is bounded. TestLocalUsernameGenericAccountNameCaughtInShellAndConfigPositions was watched failing on all sixteen positions before the change and passes after; its negative half holds prose, a longer key word and a non-account command unreported."
impact: internal
resolved_by:
  commit: "b5ab63da"
---

The generic-account floor reports a generic login only after a home root, a tilde or in an address's local part, so the account positions a shell session or a config dump writes pass unreported: a home root with no leading slash (an archive listing's Users/LOGIN/Desktop, a relative home/LOGIN/.config), a key/value position (USER=LOGIN, LOGNAME=LOGIN, username: LOGIN, login: LOGIN, a JSON "user": "LOGIN"), and the argument of an account command (su - LOGIN, chown LOGIN:staff, chown -R LOGIN). Each is where a login stands, not a word in prose, and a caller whose login is on the generic list leaks it from every one of them through capture and history. Found by the scanner-cluster review (floor gaps, item 3).

## Grounds

- pursued: a generic login in a shell or config account position is reported as local_username; one of those positions passing ScanText with no finding, or ordinary prose around the same word being flagged, would show it wrong.
