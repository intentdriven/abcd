---
schema_version: 1
id: "iss-2610111749083184"
slug: "guard-cdpath-block-reach-and-message"
severity: "minor"
category: "ux"
source: "review-followup"
found_during: "Fable review of security fix lane G, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/script.go"
remedy: "Name in the block's reason whether CDPATH was set on the line or exported in the check's environment, and resolve instead of blocking when exactly one candidate directory exists (stat cwd/name and each fixed CDPATH entry), keeping the block for the ambiguous case; prove with Check tests over a planted single-candidate and two-candidate layout"
resolution: "A cd that searches CDPATH now enters the one directory holding its operand when exactly one exists among the current directory and each fixed CDPATH entry, so the scripts after it are read there; two candidates, or an entry or current directory the line does not fix, keep the block, and its reason now says whether the command set CDPATH or the environment exported it. commands/guard.md says so. Proved by TestCDPATHSingleCandidateResolves and TestCDPATHBlockNamesWhereCDPATHCameFrom."
impact: fix
---

With CDPATH exported in the hook's environment, the new CDPATH block (iss-2610111600031244) blocks every bare `cd name`, on the line and inside every script the guard reads (`bash build.sh` whose build.sh runs `cd sub` blocks as script-unread), including where only one directory can match; and the message never says the CDPATH came from the environment, so a person whose line never mentions CDPATH is told to edit a third-party script's cd. Found by the Fable review of lane G, 2026-10-11 (cdSearchesCDPATH: if !set { return rc.files.cdpath != "" }).
