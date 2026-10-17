---
id: dcn-2608180000000001
slug: the-guard-s-parse-layer-is-a-mistake
date: 2026-08-18
---
2026-08-18 — The guard's parse layer is a MISTAKE FILTER, not a security boundary
(adr-42, iss-272). Enumeration-completeness is abandoned as the matching strategy:
Tier 1 (position-anchored, blocks) keeps the registry match; Tier 2 (position-agnostic,
warns) speculatively re-matches from each later command-position token when NO ENTRY
MATCHED, each suffix run through expandPayloads, converting the unbounded WRAPPER class
from silent allow to loud warn without enumerating anything — 6 of the 10 verified
bypasses, 7 once each speculative suffix runs through expandPayloads. It does NOT reach
the exec-string class (su/runuser/script -c keep an opaque payload isShellFamily will not
open), which stays silent until part C lands. Forced by three facts: the
defect is the wrapper path's missing fail-safe (the interpreter path has shellUnresolved,
the wrapper path has nothing), the wrapper set is unbounded in principle and its
tractable part cannot be read off the documentation (nsenter's --help renders -S as
optional-arg and unshare's renders the same letter, same package, same version, as
required-arg, while BOTH binaries consume the next token — so only probing the installed
binary is sound; gh-299 is the in-repo proof — its first list was taken from the bug
report and was wrong three ways, omitting --shallow-file (a live force-push bypass in git
since 1.9) and counting --exec-path/--super-prefix as value-taking when neither is, and
it TOTALLED NINE EITHER WAY, so a size assertion certified the wrong list as complete), and every vendor shipping this control documents it as
steering, not enforcement.
Tier 2 is BOUNDED against a measured quadratic DoS — the drafted form ran 14.9s at 6000
tokens, ~8h of CPU at the 1 MiB stdin cap, inside the PreToolUse hook — so O(N) starts, a
~64 cap, short-circuit, pinned by a benchmark. The gate is "no entry matched THIS SEGMENT" — never "argv[0]
unknown", which would switch speculation off for git, and never per-Check, which would
hand an author a one-token suppression (`git clean -fd ; nice git push --force` matches
git-clean, so a whole-command gate disarms the fail-safe for the whole line).
Per-segment is also what makes the false-positive figures a floor, but only with
"unknown command" pinned to commandOf's OUTPUT (wrappers, assignments and reserved words
stepped, basename taken) rather than the literal first token: matchSegment keys on exactly
that value, so a segment commandOf resolves to something no entry names cannot match any
entry, and the nesting is by construction. Under the literal reading it is false —
`env git clean -fd gh repo delete .` resolves to git and matches git-clean, while a
literal-token gate would see env and fire on git clean's own pathspecs.
Also decided: a synthetic verdict raised from a speculative suffix (expandPayloads emits
two blockers, envSpecialBlockSignal and depthBlockSignal) is DEMOTED to warn, never
honoured at its own tier and never dropped — honouring blocks on two layers of
uncertainty, dropping invents a second silent suppression path inside the fail-safe.
The ~64-start cap is PER SEGMENT for the same warn-rate reason. Enumeration continues,
demoted to an upgrade (warn → precise block) and safe to be incomplete. The exec-string
family (su/runuser/script/flock -c) gets a small table, NOT a generalisation of
shellCPayload, which would ship six new SILENT allows on the long spellings. Sequencing
D → A → B → C. Rejected with evidence: add-the-missing-names as the primary fix (third
instance of one defect); descend-into-any-spaced-token (breaks the 100% known-good floor —
the spc-16 incident-capture fixture fires); flag-shaped-prefix-only (silences most true
positives); allowlist (recorded for a future fail-closed mode, wrong for arbitrary developer
work). Accepted cost: false positives on unquoted hazard-shaped text wherever no entry matched —
0/1144 repo-mined lines, 21/79 adversarial, but BOTH corpora were run against an
argv[0]-gated prototype, i.e. the gate this decision rejects, so the figures are a FLOOR
(per the nesting above) with an unmeasured margin, and re-measuring under the adopted
gate is a merge precondition. The fail-safe covers the WRAPPER enumeration only: a
missing interpreter name (fish -c) and a missing git value flag are still silent allows
under Tier 2, so this record covers the predicted fourth instance only if it lands in the
wrapper set. The warn-storm STOP from the
2026-08-15 note binds: measure on real agent commands before merge.
