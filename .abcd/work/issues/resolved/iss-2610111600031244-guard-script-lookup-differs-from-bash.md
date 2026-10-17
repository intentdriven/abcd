---
schema_version: 1
id: "iss-2610111600031244"
slug: "guard-script-lookup-differs-from-bash"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "private security advisory GHSA-xfjm-pv25-v4j4, filed 2026-10-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/script.go"
remedy: "Resolve script paths physically after cd (EvalSymlinks, which is how the kernel resolves .. whatever -L or -P says), block a cd while CDPATH is set on the line or in the hook's environment unless the operand is absolute or starts with ./ or ../, and apply a source command's own PATH prefix, relative entries included, before the working directory, blocking when an entry is not fixed by the line; prove each of the three planted-tree reproductions blocks and the export PATH control still blocks"
resolution: "The script reading now resolves a .. in a path physically, as the kernel does (after cd link, ../s.sh is the file beside the link's target), and cd -P records the physical directory; a cd or pushd whose operand the shell searches CDPATH or cdpath for, while CDPATH is set on the line or in the check's environment, blocks as script-unread unless the operand is absolute or starts with ./ or ../; and a source command's own PATH prefix is the PATH its search reads, relative and empty entries resolved against the working directory, with an entry the line does not fix blocking. A shell's script operand searches the PATH the shell is handed. Proved by TestScriptPathResolvesDotDotPhysically, TestCDPATHCdBlocks and TestSourcePathPrefixIsSearched (internal/core/guard/scriptlookup_test.go), watched failing first; the export PATH control still blocks."
impact: fix
---

The guard's script reader resolves a different file than bash runs in three ways: after `cd` it keeps the logical path (so `cd -P link && bash ../s.sh` judges the file beside the link while bash runs the one beside its target), it ignores `CDPATH`, and `source` of a bare name ignores a `PATH` prefix on the same command. Each lets a planted script carrying a bundled blocker run with exit 0.

Private security advisory GHSA-xfjm-pv25-v4j4 (draft, severity high). Full text, evidence and reproduction: .abcd/.work.local/scratch/security-drain-2026-10-11/advisories/ in the main checkout's local tier. This record stays uncommitted until its fix lands; the fix commit adds it directly to resolved/.

Evidence (main 3d171421a): `cdTarget` stores `filepath.Clean` of the argument and never reads CDPATH; `resolve` joins that logical directory to a relative script; `searchPath` uses only the shell state's stored PATH, skips relative entries and falls back to the working directory (all internal/core/guard/script.go). Reproduced at that commit with `abcd guard check --command` over planted trees: `cd -P <T>/link && bash ../s.sh`, `cd <T>/empty && CDPATH=<T>/cdp cd sub && bash s.sh`, `PATH=<T>/relp source p.sh` and `PATH=relp source p.sh` are all allow (exit 0); the control `export PATH=<T>/relp; source p.sh` blocks (exit 1).
