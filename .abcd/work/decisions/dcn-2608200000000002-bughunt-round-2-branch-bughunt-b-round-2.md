---
id: dcn-2608200000000002
slug: bughunt-round-2-branch-bughunt-b-round-2
date: 2026-08-20
---
2026-08-20 — bughunt round 2 (branch bughunt-b/round-2): captured the round-2 records (iss-346..iss-357, with the two colliding ids re-minted as iss-358/iss-359), fixed
eleven records, closed iss-201, and wontfixed iss-358 on pre-merge review evidence.
Code: iss-358 lanHostRe/deviceHostRe trailing-\b redaction miss (the iss-307 sweep's
residual) — BUILT, then REVERTED: the pre-merge adversarial review proved the loosened
pattern flags snake_case selectors (stream.local_addr, args.local_rank) and Stage-1
redaction rewrites findings into the irreversible history store, so the warn-tier miss
is the accepted cost, pinned by TestSnakeCaseSelectorsStayQuiet; iss-359 redirect-controlled FinalURL
carried C1/bidi/zero-width runes raw into --json and the committed baseline (fetch
boundary percent-encodes, baseline validates, termsafe/coverage doc claims corrected);
iss-346 guard also-matched dropped a matched warn under a synthetic block; iss-347
readTranscript symlink-follow + truncation (now fsutil.ReadGuarded); iss-201 hook
stdin over-cap misdiagnosis (cap+1 probe at readHookInput and the guard hook);
iss-356 batch (urlguard RFC 6890 ranges incl. CGNAT, repolint not-scanned warn,
64-hex resolve sha, CLI help truths, release.yml wording + templates, dependabot pip).
iss-357 records the memory-ingest FinalURL sibling site the cite fix did not reach.
Records: iss-348/349/350 user-facing doc truths; iss-351 discipline roster -> pointer;
iss-352 phantom research/adr repointed; iss-353 note naming yields to adr-30; iss-354
go-version lockstep test (mutation-verified). Recorded not fixed: iss-355 — a batched
merge-queue push tags the batch tip, so auto-release + the HEAD^2^ derivation wedge a
release roll merged as a non-final queue entry (iss-326's outcome by a route itd-93
cannot close); required-check workflow logic, left with two proposed fixes. Refuted
this round (kept out of the ledger): the external-review pull_request_target check-sha
wedge (check runs attach to the PR head, proven from run history), the docs-site
ungated claim (docs-lint gates docs/ on every PR; iss-234 records the inert design),
the empty Diataxis indexes (owned by iss-216 and the facilitator plan), identity email
case-folding (byte-exact parity with the sh pre-commit gate is the contract), and the
guard backtick parser gap as a fix target (iss-148 records the reverted attempt; only
its missing help-text disclosure was new). Dedup boundary: iss-307/311/317/319/321/325
and the v0.6.0 wedge (iss-326/327) were already held by hunt A or the maintainer and
were not re-reported.
