---
schema_version: 1
id: "iss-2609251543293588"
slug: "the-generic-account-floor-misses-the-json-serialised-windows"
severity: "major"
category: "security"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/identity.go"
resolution: "accountRootPrefixes carries the doubled Windows root (\\\\users\\\\) beside the single one, and the home-literal clause of standsAsAccountName accepts the doubled spelling of a home that carries backslashes (identityMatchers.homeLiterals). TestLocalUsernameGenericAccountNameCaughtUnderAJSONEscapedWindowsRoot was watched failing on the three JSON-escaped shapes before the change and passes after, and a word after an escaped separator in prose stays vocabulary. The gap never reached a release: it came in with the generic floor in this lane."
impact: internal
resolved_by:
  commit: "0d213f8f"
---

The generic-account floor misses the JSON-serialised Windows home root. accountRootPrefixes (internal/adapter/scanner/identity.go) carries the single-backslash spelling (\users\) but not the doubled one every JSON encoder writes, so C:\\Users\\<login>\\Desktop in a transcript line raises no finding at all while C:\Users\<login>\Desktop hard-fails; the home-literal clause of standsAsAccountName compares the configured home verbatim and misses its doubled spelling the same way. Before the generic floor the bare word was flagged, so the floor narrowed a hard_fail rule in the redactor's own input shape: a caller whose login is on the generic list leaks the home path of every Windows path a JSON transcript quotes through capture and history.

## Grounds

- pursued: a generic login in a JSON-escaped Windows home path is reported as local_username; a transcript line quoting C:\\Users\\<login> that yields no local_username finding would show it wrong.
