---
id: dcn-2608200000000008
slug: bug-hunt-round-3-hunt-b-branch-bughunt-b
date: 2026-08-20
---
2026-08-20 — Bug-hunt round 3 (hunt B), branch bughunt-b/round-3. Baseline
(make preflight + gofmt -l .) green before any change. Captured iss-382..iss-392
(11 records), each adversarially refuted before capture. Fixed and resolved this
round: iss-382 disembark plan's RenderManifest printed source-repo file paths raw
while the same view sanitised its neighbours (terminal-escape injection over an
untrusted source repo); iss-383 the history transcript store's read path used raw
os.ReadFile while its write path was fully hardened, so a planted FIFO wedged
history capture under the store flock and a symlink was followed (both reads now
fsutil.ReadGuarded); iss-384 Baseline.validate sanitised final_url alone while
echoing the entry key and quoted fields raw to the terminal through the CLI error
surface (incomplete-fix residue of iss-359); iss-386 launch scaffold stripped the
go.mod patch and refused one back, shipping every adopter a floating go-version
(the adopter-side residue of iss-289); iss-385 commands/version.md still named only
two network verbs after two rewrites (docs cite refresh and memory ingest also
fetch); iss-387 the release-gate manifest pinned 17 briefDocs while four shipped
surface chapters (guard/ideate/identity/banlist) sat outside it, so every full-tier
crosscheck attested coverage it never ran; iss-388 the phases README called the
shipped itd-88 planned; iss-389 the examples-use-reserved-identifiers principle
described its shipped iss-154 lint as not-yet-built; iss-391 research/notes/README
routed to the adr-30-rejected research/phase and research/adr layouts; iss-392 the
confirmed nitpick batch (canonical id-uniqueness key closing the zero-padded-filename
hole, .yaml coverage in the go-version lockstep sweep, scripts/*.sh eol=lf, the
lifeboat-reviewer 0.1.1 reconciliation, and the commands/abcd/ path repoint across
five live records). Recorded, not fixed: iss-390 — the pre-decided promotion of the
reserved-identifiers principle to a discipline-kind intent is a maintainer ceremony,
so it stays open as the tracker. Refuted this round (kept out of the ledger): the
scanner genericHomeRe case-fold (iss-308 already adjudicated against folding the
POSIX /users/ arm — real API-route false positives, and home_path_other is warn-tier
anyway); the ahoy refounding-candidate case compare (documented mutable labels; the
miss lands on the designed decline path; iss-221 owns the real repair); govulncheck
non-gating (documented deliberate advisory, DECISIONS 2026-08-19 and commit 61a15b4);
the CODEOWNERS publish-surface omissions (the .abcd/* paths reach no installed user
and the Go-tree reading proves too much; sole-maintainer bypass makes the scenario
moot — a golden-file test for the bare scaffold render is the real follow-up if a
write rung is ever granted); isHexSHA 40-hex (display-only, sha256 repos unreachable,
iss-206 owns the sibling); the coverage_index.json raw read (already named by open
#365); commands/docs.md's scoped "on behalf of documentation" heading; the agents/
host-agnosticism claim (docs-lint bans harness names, not the model id, which is the
sanctioned disclosure token); the make smoke vacuous-gate and runbook-parity claims.
Model routing: orchestration on Claude Fable 5; five parallel hunters, seven
per-finding adversarial refuters on Claude Opus 5; the dual pre-merge review runs one
reviewer as Claude Fable 5 and one as Claude Opus 5.
