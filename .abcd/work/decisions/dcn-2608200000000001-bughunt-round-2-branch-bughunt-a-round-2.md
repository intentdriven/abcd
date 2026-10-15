---
id: dcn-2608200000000001
slug: bughunt-round-2-branch-bughunt-a-round-2
date: 2026-08-20
---
2026-08-20 — bughunt round 2 (branch bughunt-a/round-2): captured iss-331..iss-343 and fixed (ids 339-343 re-minted from an initial 326-330 after main concurrently took those numbers in the v0.6.1 cut)
ten (six fix-impact, four internal). Code/security: iss-339 the scanner's githubRemoteRe matched
the github.com host case-sensitively, so a mixed-case remote (git@GitHub.com:…) left
GitRemoteUsername empty and the github_username redaction kind never armed — the caller's handle
survived history-capture redaction and the PII scan (added (?i)); iss-340 `abcd history show`/`list`
printed the untrusted transcript body and metadata fields with no termsafe, replaying ESC/CSI/C1/bidi
sequences raw (now SanitizeBlock on the body, Sanitize on the fields); iss-341 readLifeboatFile kept
an Lstat→root.Open window over an untrusted lifeboat (FIFO hang / symlink-follow) — routed through
fsutil.ReadGuardedInRoot, dropping the unused abs param; iss-342 SourceContext.ListDir opened
directory entries with a blocking open, so a statically-planted FIFO named `docs` hung
disembark probe/plan/pack over an untrusted target repo with no race (now O_RDONLY|nonBlock, matching
ReadFile). Docs (fix): iss-343 prepare-this-repo.md named the pre-flattening commands/abcd/ path and
"three levels up" (now flat commands/ and "two levels up", tied to CLAUDE_PLUGIN_ROOT); iss-331
lint.md presented the abcd-lint:allow waiver as global when only privacy-hygiene honours it. Record
(internal): iss-332 development/README.md denied the issue ledger adr-32 created; iss-333 the brief's
05-internals/03-configuration.md development tree drew the pre-adr-30 layout (roadmap/intents, research/adr,
research/phase, .abcd/specs) — redrawn flat by artefact type; iss-334 intents/README.md taught the
v0.6.0-retired intent review verb and intent-fidelity-reviewer agent (swept to intent audit /
intent-auditor); iss-335 the issues/ store contract omitted the mandatory impact field and the
agent-observation source. Recorded not fixed (stay open): iss-336 gate_lockstep blank-path silent
disarm (nitpick, no external arming path — defence-in-depth only); iss-337 probe.go inner-descent
OpenRoot FIFO race (needs a raw openat restructure, os.Root.OpenRoot takes no flags); iss-338 seven
registry personas given gendered pronouns against the itd-79 they/them discipline (nitpick cosmetics
batch). Refuted this round: govulncheck non-gating (documented deliberate advisory — commit 61a15b4
and ci.yml:362-363), and the CHANGELOG bijection gap (the bijection reads the composer's JSON payload,
never CHANGELOG.md; the residual is the already-open iss-256). Model routing: orchestration on
Claude Fable 5; hunters, per-finding refuters, and one of the two pre-merge reviewers on Claude
Opus 5; the second pre-merge reviewer on Claude Fable 5 (dual-model merge gate).
