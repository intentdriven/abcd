---
schema_version: 1
id: "iss-308"
slug: "repolint-abspathre-windows-arm-is-case"
severity: "nitpick"
category: "bug"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: "internal/core/repolint/rule_privacy.go"
resolution: "absPathRe Windows arm is case-folded; lowercase c:\\users\\<name> is caught. POSIX arm left case-sensitive."
impact: fix
---

repolint absPathRe Windows arm is case-sensitive so a lowercase c-users path escapes the privacy rule
## Evidence
`internal/core/repolint/rule_privacy.go:57` — `absPathRe`'s Windows alternative `[A-Za-z]:\\Users\\[A-Za-z0-9._-]+` has no `(?i)`. NTFS is case-insensitive; `c:\users\bob` (Python `os.path.normcase` lowercases the whole path) is missed. The downstream `isWindowsPath` already `strings.ToLower`s, so lowercase spellings are unreachable-by-construction today.

## Adversarial verdict: CONFIRMED narrow arm only (nitpick); POSIX arm REFUTED
Folding case on the Windows arm alone adds ZERO new positives across the probe set — there is no URL-route analogue with backslashes. The POSIX `/Users//home/` arm must NOT be folded: `(?i)` there adds real FPs on API-route text (`/users/me`, `/api/v1/users/current`) at Error severity. Fix: wrap only the Windows arm `(?i:[A-Za-z]:\\Users\\[A-Za-z0-9._-]+)`. Not prior art: iss-153/154/156 are /Users/shared FPs, unrelated to case.
