---
schema_version: 1
id: "iss-2609251639261103"
slug: "home-path-other-and-home-path-self-read"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/identity.go"
deferred_after: "v0.10.0"
deferral_reason: "Not contained: a Windows spelling for home_path_other changes genericHomeRe, the path-segment byte class, the system-directory allowlist, the traversal walk and the lint audit rule that shares them, which is a lane of its own; the caller's own login in a Windows home, literal or JSON-escaped, is still reported hard_fail as local_username, so what the gap costs is the warn-level third-party path and the kind the caller's own home is reported under, not a caller leak."
resolution: "Fixed by 40136106: genericHomeRe gains the Windows alternative <drive>:\\Users\\<name> with each separator a backslash run, so the typed, JSON-escaped and doubly escaped spellings are home_path_other; the trailing boundary takes the backslash, the allowlist recognises the Windows Users root and gains Default and All beside Public (shared with the repolint privacy rule), the traversal walk reads a backslash run as one separator, and the home_path_other skip compares the caller's home against the match with runs collapsed. The caller's own home is home_path_self at any depth through the JSON-escape views of c55ae5ef (abcd builds for darwin and linux, so the caller's home is never itself a Windows path). Pinned by TestWindowsHomeIsAThirdPartyHomePath and TestWindowsOwnHomeIsHomeSelfAtAnyDepth (watched RED at 211b8853) and the false-positive guard TestWindowsSystemRootsAreNotHomes (watched failing under two mutations). The repolint privacy rule's own regexp is a twin that reads a single-backslash Windows home only; its escaped half is iss-2609261658553101. Known limit: the Default and All allowlist entries also apply to the POSIX Users root and to the repolint privacy rule, so a macOS account literally named Default or all raises no home_path_other, the WARN-level third-party kind; the caller's own login is still reported hard_fail as local_username."
impact: fix
resolved_by:
  commit: "40136106"
---

home_path_other and home_path_self read no Windows home spelling beyond the literal one. genericHomeRe (internal/adapter/scanner/identity.go) is POSIX-only, so C:\Users\OTHER\Desktop and its JSON-escaped spelling C:\\Users\\OTHER\\Desktop raise no home_path_other, and home_path_self matches the configured home verbatim, so the escaped spelling of the caller's own Windows home is reached only through local_username on its last segment. A Windows spelling threads through genericHomeRe, the path-segment byte class, the system-directory allowlist, the traversal walk and the lint audit rule that shares them, so it is not a contained change. <!-- abcd-lint:allow -->

## Grounds

- pursued: a third party's Windows home, typed or escaped, is reported and masked, and a Windows system profile directory is not; an escaped Windows home of a third party reaching the store verbatim, or C:\Users\Public reported as a user, would show it wrong
