---
schema_version: 1
id: "iss-272"
slug: "the-guard-s-wrapper-set-has-the-same"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
resolution: "adr-42 built in four parts. D: the guard's three surfaces now state that the parse layer is a mistake filter, not a security boundary, and name the execution layer as where an enforced boundary lives. A: a per-segment fail-safe — when no entry matched a segment the matcher re-runs from each later command-position token and each suffix goes through expandPayloads; a hit is a loud warn, never a block, because an unknown command is not proof it execs its arguments. Bounded twice: 64 starts and 512 tokens per start, after capping starts alone still measured 14.2s on a 1 MiB line. B: fourteen wrapper names, every flag list derived by probing the installed binary rather than reading --help, which caught two table errors before they shipped. C: an exec-string table for su/runuser/script/flock, covering the six long spellings a shellCPayload generalisation would have silently allowed. All ten recorded bypasses now produce a verdict; 961 mined command lines and a labelled adversarial corpus ship as test data."
impact: fix
resolved_by:
  commit: "01b9328"
---

The guard's WRAPPER set has the same incomplete-enumeration shape gh-297 fixed for interpreters. gh-297 made the execute-a-string INTERPRETER family one shared predicate (sh, bash, dash, zsh, ksh, mksh, ash). The wrappers map in internal/core/guard/match.go is a separate list of commands that merely exec another, and it omits several that are equally ordinary: nice (and 'nice -n 5'), setsid, flock, stdbuf, ionice, unshare, chroot, and busybox sh -c / busybox ash -c. Each is a silent allow today for every bundled blocker -- and equally so before gh-297 with plain sh, so none is introduced by that change. Also absent from BOTH sets: su -c '<hazard>', runuser -c '<hazard>' and script -c '<hazard>' /dev/null, which are execute-a-string interpreters in exactly the sense gh-297's rationale describes (each runs its -c operand with the grammar the tokenizer already parses) but are not shells, so they belong in neither list as currently shaped. The realistic ones are nice, setsid, busybox sh and su -c. Out of scope by contrast: ssh host '<cmd>' and docker run img sh -c '<cmd>' execute on a different host or namespace. Note for whoever takes this: gh-297's comment now says explicitly that 'nowhere left to widen' covers the interpreter set only, not the wrapper set, so the claim and this gap do not contradict each other. Found by the security review of gh-297.