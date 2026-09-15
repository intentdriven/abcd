---
schema_version: 1
id: "iss-2609012039103376"
slug: "ghsa-m8pg-doctor-json-leaks-home-paths-in-path-stale-detail"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/apply.go"
resolution: "auditGaps renders both paths in the history.path_stale detail through displayPath, so a registered path and a current path under the home directory reach doctor's JSON in tilde form; TestAuditGapsRedactHomeInStalePathDetail (core) and TestAhoyDoctorJSONCarriesNoHomePrefix (cli surface) pin it, the latter also asserting the whole doctor --json output carries no home prefix. Sibling sweep: every other path-bearing gap already routes through displayPath or uses a literal ~/.abcd/history prefix; guard_health's registry reason is a constant string. A foreign machine's home in the registered path is out of scope and left as recorded."
impact: fix
---

GHSA-m8pg-chhv-hxvq (CWE-200, advisory severity low): `ahoy doctor --json` leaks absolute home paths in the `history.path_stale` gap. `internal/core/ahoy/apply.go:auditGaps` composes the gap's Detail from two raw absolute paths — `entry.Path` from `~/.abcd/history/index.json` and `filepath.Abs(cwd)` — while every other path-bearing gap in the package (`detectPathSymlink`, `detectShadowedEntry`, `detectBinDirOnPath`) routes through `displayPath` (`fsutil.RedactHome`) precisely so a gap pasted into an issue carries no username. Text-mode doctor prints counts only; `--json` encodes the whole DoctorReport raw. Reproduced at v0.7.0 (8f68ffb3): register a repo, move it, `ahoy doctor --json` prints both paths in full. The fix must establish that both paths render through `displayPath` and that doctor's JSON output carries no home prefix; a foreign machine's home in `entry.Path` (a different username in a cross-machine registry) is not this advisory's scope.

## Grounds

- pursued: routing the two paths through the package's one redactor is the whole fix; the surface test proves no other field of doctor's JSON leaks the home prefix, which would show the fix incomplete
