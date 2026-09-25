---
schema_version: 1
id: "iss-2609251639024615"
slug: "the-account-key-walk-gives-up-after-eight-blanks-which-fails"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/identity.go"
resolution: "maxKeyGap (internal/adapter/scanner/identity.go) is 32 blanks, so afterAccountKey and afterAccountCommand read a column-aligned gap between USER=, username: or su and the login as the account position it is, still a constant read per match. TestLocalUsernameGenericAccountNameCaughtInShellAndConfigPositions was watched failing on ten-, twenty- and twelve-blank gaps before the change and passes after. The bound came in with the shell and config positions in this lane and never reached a release."
impact: internal
resolved_by:
  commit: "24a8954e"
---

The account-key walk gives up after eight blanks, which fails away from the finding. maxKeyGap (internal/adapter/scanner/identity.go) bounds the blanks afterAccountKey reads between a key and its value at eight, so a column-aligned config dump (USER= or username: followed by ten blanks and the login) raises no finding for a login on the generic list. Every other bound in the position checks stops reading and keeps the match; this one stops reading and drops it.

## Grounds

- pursued: a generic login after an account key or command separated by up to maxKeyGap blanks is reported as local_username; a column-aligned USER= or username: line within the bound that yields no local_username finding would show it wrong.
