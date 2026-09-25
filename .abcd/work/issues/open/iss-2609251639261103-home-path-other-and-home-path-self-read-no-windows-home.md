---
schema_version: 1
id: "iss-2609251639261103"
slug: "home-path-other-and-home-path-self-read-no-windows-home"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/identity.go"
deferred_after: "v0.10.0"
deferral_reason: "Not contained: a Windows spelling for home_path_other changes genericHomeRe, the path-segment byte class, the system-directory allowlist, the traversal walk and the lint audit rule that shares them, which is a lane of its own; the caller's own login in a Windows home, literal or JSON-escaped, is still reported hard_fail as local_username, so what the gap costs is the warn-level third-party path and the kind the caller's own home is reported under, not a caller leak."
---

home_path_other and home_path_self read no Windows home spelling beyond the literal one. genericHomeRe (internal/adapter/scanner/identity.go) is POSIX-only, so C:\Users\OTHER\Desktop and its JSON-escaped spelling C:\\Users\\OTHER\\Desktop raise no home_path_other, and home_path_self matches the configured home verbatim, so the escaped spelling of the caller's own Windows home is reached only through local_username on its last segment. A Windows spelling threads through genericHomeRe, the path-segment byte class, the system-directory allowlist, the traversal walk and the lint audit rule that shares them, so it is not a contained change.
