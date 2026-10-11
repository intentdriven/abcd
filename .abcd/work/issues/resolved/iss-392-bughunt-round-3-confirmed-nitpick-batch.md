---
schema_version: 1
id: "iss-392"
slug: "bughunt-round-3-confirmed-nitpick-batch"
severity: "nitpick"
category: "bug"
source: "agent-finding"
found_during: "bughunt-round-3"
found_at: ".abcd/work/issues"
resolution: "confirmed nitpick batch: canonical id-uniqueness key, .yaml lockstep sweep, scripts eol=lf, lifeboat-reviewer 0.1.1, commands/abcd path repoint"
impact: fix
---

bughunt round-3 confirmed nitpick batch — zero-padded record-id filenames evade the id-uniqueness lint in all three families; the go-version lockstep test skips .yaml workflows; scripts/*.sh lack the eol=lf pin the repo's own policy states; agents/CHANGELOG announces lifeboat-reviewer 0.1.1 while the prompt carries 0.1.0; five live records still route to the pre-flattening commands/abcd/ and retired docs/reference pages
## Evidence

Five confirmed nitpicks, batched per round precedent (iss-356):

1. Zero-padded record ids evade the id-uniqueness lint in all three families — `internal/core/lint/lint.go:72` (`issueIDRe = iss-\d+`) keys `validateIDUnique` (`:1667`) on the raw filename match, so `iss-0100-…md` beside `iss-100-…md` yields two keys and no finding (probe: record-lint exit 0 with a planted `id: "iss-100"` collision), while `recordid.fileID` (`internal/core/recordid/resolve.go:176-178`) deliberately normalises the parsed integer and `record_schema`'s filename check (`internal/core/lint/schema.go:259-285`) compares parsed handles — the backstop's hole is exactly the hand-added case its own doc-comment (`lint.go:1770-1776`) says it exists to close. Consequences narrowed by the refuter: the only live `NewResolver` consumer checks existence only, and the auto-allocator parses integers, so no runtime misbehaviour today.
2. `goversion_lockstep_test.go:28` skips any workflow not ending `.yml`, so a `.yaml` workflow (or `go-version-file:`) escapes the lockstep sweep; the `checked == 0` tripwire catches only a total miss.
3. `.gitattributes` pins `eol=lf` for `.githooks/*`, `internal/core/ahoy/defaults/*` and `hooks/*.sh` with a CRLF-shebang rationale, but not `scripts/*.sh` — on an autocrlf-on checkout the pinned pre-push hook survives and then dies one hop later in the unpinned `scripts/check-reviews.sh` (`set: pipefail: invalid option name`, verified). CI is unaffected (ubuntu runners checkout LF); the dev-machine path is the exposure.
4. `agents/CHANGELOG.md:130` announces `lifeboat-reviewer 0.1.1 (renamed from lifeboat-oracle)` while the prompt carries `prompt_version: 0.1.0` at `agents/lifeboat-reviewer.md:4,53,70` and in its canary fixture — the CHANGELOG is the sole outlier (`agents/README.md:45`'s "all 0.1.0" agrees with the files); no consumer pins a value, so the file-side bump is the safe alignment.
5. Pre-flattening path residue in live (non-historical) records — `commands/abcd/intent.md` and retired `docs/reference/{commands,facilitator,review-schema}.md` names survive in `intents/planned/itd-43` (`:43,:73`), `itd-48`, `itd-2`, `intents/drafts/itd-85`, and `specs/open/spc-6` — outside the DECISIONS.md:919 historical carve-out (decisions, shipped intents, closed specs, research, resolved issues). `brief/02-constraints/04-naming.md:74` cites `docs/reference/review-schema.md`, which exists nowhere; that citation needs a maintainer's judgement on the intended target and is recorded, not repointed.

Refuter verdicts: 1-5 CONFIRMED nitpick (item 1 probe-proven; item 3's CI framing and item 5's itd-43-only framing corrected). REFUTED alongside and not carried: the `make smoke` vacuous-gate claim (a lost build tag runs more, not less; a renamed harness exits 1) and the runbook-parity claim (the launch.md sentence scopes itself to the workflows; runbook byte-parity is impossible by design and its gate list is pinned in both profiles).
