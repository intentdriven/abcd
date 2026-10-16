---
id: dcn-2608180000000002
slug: adr-42-parts-d-and-a-built-two
date: 2026-08-18
---
2026-08-18 — adr-42 parts D and A built. TWO amendments the build forced, both recorded
in the ADR. (1) The ~64-start cap does NOT bound the cost on its own: every speculative
start pays commandOf, which walks the whole leading wrapper chain, so 64 starts on a 1 MiB
line of `env env env ...` measured 14.2s inside the PreToolUse hook — linear in line
length and still an outage. A second bound (maxSpeculativeWindow = 512 tokens read per
start) takes it to ~120ms; a truncated window warns through the same fail-loud path as an
exhausted start cap, so the coverage trade is never silent. Pinned by a test at the real
1 MiB stdin cap, watched failing at 14s with the window removed. (2) A speculative start
must NOT skip known wrappers, though commandOf would step them: the wrapper is exactly
where a payload lives, and `myrunner env -S '<hazard>'` needs `env` in command position for
expandPayloads to read the -S value at all. Skipping wrappers is lossless for MATCHING and
lossy for EXPANSION — caught by the demotion test, which failed as an allow.
The warn-storm STOP is DISCHARGED with committed evidence: 961 mined command lines with
zero Tier 2 fires, plus a labelled adversarial corpus (23 quiet / 18 caught / 5 Tier 1
blocks / 2 accepted false positives), both under internal/core/guard/testdata/corpus/ with
the ceiling enforced by a test. The shipped fail-safe is much quieter than the
argv[0]-gated prototype predicted, because a hazard inside a quoted argument or a path
never reaches command position. `git bisect run` and `git submodule foreach` now warn,
exactly as the not-argv[0] gate predicted.
