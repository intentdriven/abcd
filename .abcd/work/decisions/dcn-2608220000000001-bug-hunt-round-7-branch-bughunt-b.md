---
id: dcn-2608220000000001
slug: bug-hunt-round-7-branch-bughunt-b
date: 2026-08-22
---
2026-08-22 — Bug-hunt round 7, branch bughunt-b/round-7. Baseline (make preflight
+ gofmt -l .) green before any change and on the final tree; started from main at
9db1316. Five parallel Opus 5 hunters swept the four dimensions; each candidate was
adversarially refuted by an independent Opus 5 subagent before capture. Findings:
8 substantive (2 major, 6 minor) + 8 nitpick confirmed, 7 refuted/prior-art;
nitpicks-only: no. Fixed and resolved this round, each behaviour-changing code fix
carrying a test watched fail before the change and pass after (the repolint size
TOCTOU excepted — closed by construction, its boundary test guards the cap not the
race): iss-2608220131352917 the guard tokenizer read a leading & as a
background/&& operator, so a glued or spaced &>/&>> both-streams redirection split
the simple command and dropped its dangerous flag out of command position — a
silent allow on every blocker-tier entry; now recognised as a redirection before
the list split, target dropped, fd digit kept. iss-2608220134344680 the frontmatter
scanners anchored on line 0 and TrimSpace does not strip a UTF-8 BOM, so a BOM- or
multi-line-comment-led record slipped the no_git_metadata blocker and the
record_schema gates; shared frontmatter.TrimBOM plus a stateful multi-line-comment
skip in the lint and glossary scanners. iss-2608220142158516 abcd update and abcd
history leaked the absolute home root into their success/refusal envelopes (text and
--json), including the plugin-session refusal the plugin relays into agent chat;
redacted to ~ via a new fsutil.RedactHome at the render boundary. iss-2608220136593438
ahoy tested .git for dir-ness, so a linked worktree or submodule (gitfile .git) was
misclassified as an unmanaged folder and ahoy install exited 0 aborted with a wrong
reason; now tests existence, the last isDir(.git) holdout after iss-72.
iss-2608220136597127 ahoy --json serialised a never-computed all-false guard object
for an unmanaged folder; the field is now a pointer omitted for a folder, matching
the Banlist sibling. iss-2608220144233519 the repolint privacy scanner read exactly
the cap, so a file grown past it during the read was scanned as a truncated prefix
and reported clean; now cap+1 with the grown file routed to the not-scanned path.
iss-2608220147106835 spec/intent/memory/capture emitted bare null for an empty --json
collection; the constructors seed them non-nil so an empty store marshals [].
iss-2608220150151397 the mental-model brief claimed zero live bundles while four
planned intents declare a bundle-member of spc-83 (residual of iss-123's README-only
fix). Nitpicks: iss-2608220142154022 update.Plan had no default so an unknown target
kind fell through to swap (now fail-closed); iss-2608220145356167 docs lint returned
an engine fault as exit 1 not 2; iss-2608220148289898 abcd adr-N routed by a fixed
%04d filename prefix, missing a differently-padded ADR file (spc-26 amended);
iss-2608220149008905 the pull_request_target checkout guard external-review.yml states
in prose was unarmed (now a workflow-walking test); plus the doc corrections
iss-2608220150157497 (adr-22/26 links), iss-2608220150154972 (version.md install_mode),
iss-2608220150152332 (CI classifier standdown), iss-2608220150152535 (docs
requirements transitives). Refuted / prior art (kept out of the ledger): the abcd-cli
name in commands/launch.md (the live internal project name, record-lint-prescribed);
the spc-3 predecessor-store collision in intents/README (a new instance of open
iss-239, carried there); the .abcd/work roster intake.md/attribution-rewrite omission
(round 4's highlight-not-inventory doctrine still governs); the CI gitleaks same-origin
checksum (adr-46's accepted trust bar, residual tracked as open iss-379); the go.mod
vs setup-go patch coupling (GOTOOLCHAIN=auto self-corrects); the Makefile CRLF eol
pin (GNU make strips the CR on POSIX); and the wrangler/mkdocs docs-site deploy pin
(dashboard-side, not an in-repo mechanism). Model routing: orchestration on Claude
Fable 5; five parallel hunters and twenty-two per-finding adversarial refuters on
Claude Opus 5; the dual pre-merge review runs one reviewer as Claude Opus 5 and one
as Claude Fable 5.
