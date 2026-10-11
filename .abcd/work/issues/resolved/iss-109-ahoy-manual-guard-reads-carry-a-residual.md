---
schema_version: 1
id: "iss-109"
slug: "ahoy-manual-guard-reads-carry-a-residual"
severity: "minor"
impact: fix
category: "observation"
source: "user-observation"
found_during: "manual-capture"
resolution: "Routed the three residual Lstat->ReadFile ahoy reads (verifyHookManifest, gitignore.go x2) through fsutil.ReadGuarded — single O_NOFOLLOW|O_NONBLOCK open validating regular-file+size on the read fd — closing the read-time TOCTOU. Sentinels mapped back to existing signals: verifyHookManifest reason strings and the .gitignore refusals preserved unchanged; distinct symlink pre-checks kept as belt-and-suspenders. Structural detector (no bare os.ReadFile in ahoy) watched fail->pass; per-site FIFO/oversize signal tests added. Sibling of iss-97."
---

ahoy manual-guard reads carry a residual Lstat->ReadFile TOCTOU: verifyHookManifest (store.go), and gitignore.go read sites (~72, ~181) each do Lstat+IsRegular+size-cap then a separate os.ReadFile, so a type/symlink swap between check and read is not refused on the same fd. iss-97 converged the six bare-unguarded ahoy reads onto fsutil.ReadGuarded (single-fd O_NOFOLLOW|O_NONBLOCK+cap); these three were left because their structured signals (verifyHookManifest reason strings; gitignore bool flags + the write-path oversize refusal) need preserving, so the convergence is not mechanical. Route them through ReadGuarded too, preserving those signals, to close the TOCTOU and reach one guarded-read mechanism. Sibling of iss-97.

Priority (iss-97 security review): the static-plant FIFO-hang and unbounded-read (iss-97's actual severity) are already closed at all three sites by the pre-existing Lstat+IsRegular guard — only the narrower race-window swap remains. Of the three, gitignore.go (72/181) reads cwd/.gitignore, the SAME attacker-influenced boundary as iss-97, so it is the more exposed and should be prioritised; verifyHookManifest reads pluginRoot/hooks/hooks.json (ABCD_PLUGIN_ROOT/executable-dir sourced — the trusted install location, not cwd), a much weaker boundary, so it is lowest priority.