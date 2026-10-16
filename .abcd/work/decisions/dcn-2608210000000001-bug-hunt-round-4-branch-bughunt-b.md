---
id: dcn-2608210000000001
slug: bug-hunt-round-4-branch-bughunt-b
date: 2026-08-21
---
2026-08-21 — Bug-hunt round 4, branch bughunt-b/round-4. Baseline (make preflight
+ gofmt -l .) green before any change. Five parallel Opus 5 hunters swept the four
dimensions; each candidate was adversarially refuted by an independent Opus 5
subagent before capture. Findings: 8 substantive, 7 nitpick, 2 refuted. Fixed and
resolved this round (code, each with a watched-fail test): iss-2608211132061930
(MAJOR) lint.LoadConfig read the docs-lint/record-lint config with a raw os.ReadFile
while its .abcd/*.json siblings guard.Load/rules.Load/positioning.LoadConfig all
guard it — hook-reachable (ahoy.Detect) and cross-repo-clonable, so a committed
config symlink to a FIFO wedged docs lint/lint/ahoy/hook/cite refresh, /dev/zero
OOMed the CLI, and an out-of-repo target ran the whole ruleset from a file the repo
does not own; now fsutil.ReadGuarded + an Lstat dir-symlink refusal. iss-2608211134281912
four stdin operand readers (ideate/lessons/synthesis/readSource) read exactly cap
and silently truncated an over-cap payload — worst on history capture (stored an
8 MiB prefix under a sha256 idempotency key over the prefix, breaking spc-4's
refuse-whole invariant); added readCappedStdin (cap+1 probe). iss-2608211139076040
the iss-392 canonRecordID fix left four sibling lint keyspaces raw (delivery_state
bucket + citation, spec intent-existence, superseded_by, spec_id_unique), so a
zero-padded id slipped a gate open or false-blocked a link; routed every key and
lookup through canonRecordID. iss-2608211140410761 abcd adr-N confirmed the
frontmatter id byte-exact while record-lint and the citation resolver treat
padded/quoted/cased ids as one handle, reporting a present ADR absent (and
abcd adr-0003 routing then refusing); now a parsed-handle compare rendering the
canonical id. Docs/record fixes: iss-2608211142146585 + iss-2608211142142469 the
release-gate runbook claimed the verify job runs macOS+Linux (it is ubuntu-only;
ci.yml's check job is the matrix and the merge gate) and framed the semantic-receipt
gate as dormant-until-public-flip though it is public and has fail-closed two releases
(v0.3.0/iss-108, v0.6.0/iss-326). iss-2608211142496517 cmd/scaffold-sync named a
"scaffold-sync workflow" that was built and rejected (iss-209). iss-2608211143184945
+ iss-2608211143185943 the README Go badge (1.25→1.26) and the retired "next free id"
capture description. iss-2608211144265046 commands/lint.md documented a line:0 sentinel
the omitempty JSON never emits. iss-2608211144555085 the ADR index omitted the accepted
adr-45 row (third recurrence of the class; iss-38 wants such indexes gated). iss-2608211145448208
the DCO deferral text adr-43 retired survived in AGENTS.md and scripts/check-attribution.sh.
iss-2608211146222734 the banned pre-adr-30 "roadmap/intents" literal survived in AGENTS.md
and .abcd/README.md, outside the lint roots. iss-2608211146562243 RD001 was documented
unconditional though check-reviews.sh exempts 7 of 10 live dirs (sha-keyed receipts).
iss-2608211147403129 the ADR-RFC frontmatter pairing contract disagreed across three
surfaces; reconciled rfcs/README.md and rfc-2. Refuted (kept out of the ledger): the
CHANGELOG [Unreleased] Fixed-vs-additive sectioning (the cut discards Unreleased prose
and re-derives sections from records; owned by open iss-256); the .abcd/work tier map
"omission" (highlight enumerations, not inventories; iss-333 already blessed the table);
and the launch DenyNamespaces case-fold symlink slip (inert — the hardlink alias map
rejects via aliasDenied). Model routing: orchestration on Claude Fable 5; five parallel
hunters and ten per-finding adversarial refuters on Claude Opus 5; the dual pre-merge
review runs one reviewer as Claude Fable 5 and one as Claude Opus 5.
