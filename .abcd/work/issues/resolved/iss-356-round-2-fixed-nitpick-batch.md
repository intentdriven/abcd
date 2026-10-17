---
schema_version: 1
id: "iss-356"
slug: "round-2-fixed-nitpick-batch"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "bughunt-round-2"
found_at: "internal/surface/cli/guard.go"
resolution: "all thirteen batch items fixed: guard help backtick disclosure, urlguard reserved ranges, hook cap probe, repolint not-scanned warn, 64-hex commit sha, launch Short, release.yml wording plus templates, dependabot pip, chapter rosters, narration blocks, phantom ledger cite, activity-record noun"
impact: fix
---

round-2 fixed nitpick batch: guard check help omits the disclosed backtick limit; urlguard misses CGNAT 100.64/10 and other RFC 6890 ranges; readHookInput misreports an over-cap payload as malformed (iss-201's class at the shared helper); repolint privacy-hygiene silently skips textual files over the 4 MiB cap; capture reCommitSha refuses 64-hex commits; launch Short hides that --dry-run is required; release.yml verify-tag comment overstates what gh verifies; release.yml header says dispatch runs only the rehearsal job; dependabot covers no pip ecosystem for docs; brief pointer lines enumerate five of nine internals chapters; phase-6 and 06-lint carry change-narration blocks; 05-internals cites a ledger entry that does not exist; 03-evidence names the retired activity record nitpick documentation
## Evidence (per item; all verified by independent refuters)

1. guard help: `internal/surface/cli/guard.go:61-76` gap list omits the backtick limit that `internal/README.md:60-62` and [dcn-2608010000000004](../../decisions/dcn-2608010000000004-guard-batch-iss-159-144-148-revert-of.md) claim is stated there (the parser gap itself is iss-148, open, deliberately deferred after a reverted fix).
2. urlguard: `internal/urlguard/urlguard.go:32-45` misses RFC 6598 100.64.0.0/10 (Tailscale/AWS-internal; verified no predicate catches it, NAT64 unwrap re-checks the same list) plus 198.18.0.0/15, 192.0.0.0/24, 240.0.0.0/4, 0.0.0.0/8; doc-comment overclaims.
3. `readHookInput` (`internal/surface/cli/cli.go:950`) lacks the cap+1 probe, so an over-cap prompt-router payload is misreported as malformed JSON (fails closed+loud; only prompt-router is reachable — the other hook payloads are path-sized). Same class as iss-201 at the shared helper; the fix routes `guard.go:169` through the same probe and closes iss-201.
4. repolint privacy-hygiene: `rule_privacy.go:275-277` returns not-ok for >4 MiB files and the caller bare-continues (`:102-106`) — "conforms" without scanning, against the engine contract at `repolint.go:93-94`. Warn only for textual files (first-8-KiB isBinary probe) so binary assets stay quiet.
5. `capture.go:239` `reCommitSha {7,40}` refuses a 64-hex commit; the sibling `lint.go:78` already accepts `{7,64}`; spc-25's rationale (never refuse a legitimate resolution) argues for widening. The value is a provenance stamp, not a filter.
6. `cli.go:243` launch Short reads as optional while bare `abcd launch` refuses; the `cli.go:2099` list Short is the house precedent for naming a required flag.
7. `release.yml:307-308` comment claims `--verify-tag` checks the tag points at the checked-out commit; gh checks existence only, as `release.yml:22-23` itself states. Same false conjunct twice in `scaffold/templates/release.yml.tmpl`. Comment fix only; no assertion step (tag-move is governed elsewhere; a step would imply a guarantee it cannot give).
8. `release.yml:36` "runs ONLY the rehearsal job" is false (`verify` has no `if:` and runs on dispatch too); `:453` "full gate armed" overstates the two commands the job runs — align with `:440`'s own hedge. Mirrored in the template.
9. `dependabot.yml` declares gomod + github-actions only; `docs/requirements.txt` has no pip ecosystem; gitleaks/zizmor pins are hand-reviewed with no comment saying so.
10. brief pointer lines enumerate five of nine 05-internals chapters (`brief/README.md:26`, `brief/04-surfaces/README.md:54`; the tree line and the 05-internals index are the single rosters to keep).
11. change-narration blocks: `roadmap/phases/phase-6-lifeboat.md:19-33`, `:79-84`, `:131-138`, and the "formerly pending here" parenthetical at `brief/05-internals/06-lint.md:35` (rule at `development/README.md:25`; content decision-anchored, rephrase only).
12. `brief/05-internals/README.md:21` cites a ledger entry "[Brief restructure — future enforcement note]" that has never existed in any tree (structurally impossible in the iss-N store).
13. `brief/03-evidence/README.md:6-7` names "the activity record", retired by adr-32; the sibling chapter already says "the working record under `.abcd/work/`".
