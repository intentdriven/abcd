---
id: dcn-2608210000000003
slug: bug-hunt-round-5-branch-bughunt-b
date: 2026-08-21
---
2026-08-21 — Bug-hunt round 5, branch bughunt-b/round-5. Baseline (make preflight
+ gofmt -l .) green before any change. Five parallel Opus 5 hunters swept the four
dimensions; each candidate was adversarially refuted by an independent Opus 5
subagent before capture. Findings: 4 substantive, 9 nitpick, 9 refuted. The C1
ScrubbedEnv/GIT_CONFIG_GLOBAL candidate split the refuters (one CONFIRMED, one
REFUTED); adjudicated REFUTED — keeping the developer's global git config readable
is deliberate and test-pinned (TestScrubbedEnvStripsHijackKeepsGlobalConfig), the
attack needs control of abcd's process env (out of scope under the trusted-env
model), and the obvious fix (scrub to /dev/null) would blind the redaction probe
for every developer whose identity lives in global config. Fixed and resolved this
round (code, each with a watched-fail test): iss-2608211432258689 launch --dry-run
printed WouldRefuseOn unsanitised so a committed control-char filename injected raw
terminal escapes into the preview and CI log (now termsafe); iss-2608211432257954
the same dry-run leaked a raw absolute path in lockstep.detail (a success envelope
that never passes the error-surface scrub) — stripped at loadJSON; iss-2608211432258975
abcd lint mapped a rule-engine fault to exit 1 (the tri-state's warnings-only code)
so a CI gate keying on >=2 read a lint that never ran as an advisory pass — now
exit 2; iss-2608211432254405 launch scaffold deriveBranch assumed .git is a
directory, stamping the fallback branch main into the generated release workflows
from a linked worktree — now resolves the gitfile (worktree gitdir HEAD + shared
commondir origin/HEAD). Nitpicks fixed: iss-2608211432389181 the intent verdict
reader was the last CLI operand on Lstat-then-os.ReadFile (false benign-TOCTOU
comment) — routed through fsutil.ReadGuarded; iss-2608211432384430 the history
rootSHA diagnostic named only 40 chars though the regex accepts 64 (SHA-256);
iss-2608211432384091/389477/389791 three user-facing doc claims (README SessionEnd
self-bootstrap overclaim, memory.md source.class vs source_class, ahoy.md missing
the shadowed-on-PATH install_mode suffix); iss-2608211432489481/489288/482363/483805
record hygiene (seven display-text/href link mismatches, two bare paths to absent
dirs, the rfc-1/itd-26 one-sided pair, the work-tier roster omitting the issue
ledger/reviews/rulesets). Refuted / prior art (kept out of the ledger): C1 above;
C2 folding local_username to case-insensitive (net-negative — broadens a documented
hard_fail false-positive class and reopens iss-31); S3 widening hasControlChar (the
render sanitiser, not the integrity gate, is the fix); S4 launch --no-index dropping
a force-added tracked file (dry-run preview writes no artefact, exclusion is visible
and over-exclusion is the safe direction); D2 guard check exit-2 enumeration (the
governing "could not be evaluated at all" definition already covers a disabled
registry); R2 itd-3 spc-1 (a documented, minting-honored reservation). Model routing:
orchestration on Claude Fable 5; five parallel hunters and nine per-finding
adversarial refuters on Claude Opus 5; the dual pre-merge review runs one reviewer
as Claude Opus 5 and one as Claude Fable 5.
