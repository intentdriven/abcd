# Plugin-user safety — security and bug-fix plan and run queue (2026-08-15)

**Status:** the first of the three forward plans settled at the 2026-08-15
maintainer grill, and the current execution priority. Consumed by the generic
protocol at [`2026-07-12-abcd-run-protocol.md`](2026-07-12-abcd-run-protocol.md).
This file owns the pick-up order for
[`2026-08-11-install-experience.md`](2026-08-11-install-experience.md) Cut B
(referenced, never copied — where the two disagree, that plan and itd-108 win)
and absorbs the security remainder of
[`2026-07-29-v0.5.0-security-and-consistency.md`](2026-07-29-v0.5.0-security-and-consistency.md).

**Admission test.** An item qualifies only if a plugin installer can reach it
on a supported flow — installing the plugin, opening a session in a managed
repo, or running a documented verb. Maintainer-machine-only friction fails the
test and belongs in
[`2026-08-15-predictable-development.md`](2026-08-15-predictable-development.md);
anything the facilitator merely *reads or invokes* belongs in
[`2026-08-15-facilitator-experience.md`](2026-08-15-facilitator-experience.md).
The ledger stays the backlog of record: an issue not named here is not lost,
it just has not passed this test yet.

**Framing (maintainer grill, 2026-08-15).** The repo is public and plugin
installs are live, so defects reachable by an installer are the floor nothing
else stands on. Priority among the three plans is this plan first, then
predictable-development, then facilitator-experience — settled at the grill
against the listed 1-2-3 order.

## Run contract

Identical to the install-experience plan's (gate `make preflight` plus
`gofmt -l .` before pushing; one item per burst; kernel-format
`Assisted-by:` trailer; `abcd:ruthless-reviewer` on every item;
`abcd:security-reviewer` on every trust-boundary diff — which in this plan is
every Workstream C and D item by definition; strike limit 3; one PR per item;
the fixing PR moves the ledger entry open → resolved). Auto-merge is **not
inherited**: the maintainer authorises or withholds it when a cycle starts.

Execution mode is marked per item. *Autonomous-eligible* means the item meets
the curation bar (vetted body, armed detector, no open design question, no
collision); everything else is *human-paired*.

## Workstream A — the install is small (Cut B, referenced)

The install-experience plan's Cut B (B1 payload asset, B2 plugin-source
repoint, B3 skew-notice retirement resolving
[iss-206](../../work/issues/wontfix/iss-206-the-version-skew-notice-promised-by-itd-105-given-an-install.md),
B4 record consequences) is this plan's opening workstream. Its §4 manual
verification gate remains the prerequisite; its collision notes and the
itd-108 precedence rule apply unchanged. Nothing is re-specified here —
this plan only says: Cut B before Workstream C's tail, because a small,
proven install path removes a whole class of installer-reachable failure at
once. Human-paired (the §4 gate is manual by design).

## Workstream B — intent milestone

- **[itd-111](../intents/planned/itd-111-a-stale-abcd-never-answers-silently-every-surface-that-runs.md)
  is planned (interview run 2026-08-15, spc-22, `intent ready` exit 0).**
  The interview ran the itd-84 decomposition (SPLIT: network posture →
  adr-38 + brief invariant 7) and the SOTA fit-challenge (path 2 UPHELD);
  the micro-prompt open question was struck to
  [iss-230](../../work/issues/open/iss-230-one-tap-micro-prompt-channel.md).
  Next step here: implement against spc-22 when this plan's fix queue
  permits.

## Workstream C — fix queue (sequenced)

1. **[iss-200](../../work/issues/resolved/iss-200-guard-env-split-string-glued-form-bypass.md)**
   (critical) — the `env -S`/`--split-string` guard bypass. Autonomous-eligible:
   the repair design is agreed and recorded (handle the glued
   `-S<value>`/`--split-string=<value>` forms alongside the separate-token form
   in `commandOf`/`skipWrapperArgs`; no nested re-tokenising). The prior
   attempt's first-attempt BLOCK stands as the review bar: all three spellings
   or nothing.
2. **[iss-202](../../work/issues/resolved/iss-202-scanner-pii-config-unguarded-read.md)**
   (critical) — `scanner.New` reads `pii.json` unguarded. Autonomous-eligible.
3. **[iss-203](../../work/issues/resolved/iss-203-audit-privacy-degraded-scanner-silent.md)**
   (major) — audit silently degrades when `scanner.New` fails. Same seam as
   iss-202; land with or immediately after it, never in parallel.
4. **[iss-201](../../work/issues/resolved/iss-201-guard-hook-stdin-overflow-fail-open.md)**
   (major) — guard hook fails open past 1 MiB of stdin. Autonomous-eligible.
5. **[iss-210](../../work/issues/resolved/iss-210-lone-token-subverb-guess-writes-a-record.md)**
   (major) — a lone mistyped token writes a ledger record. Autonomous-eligible.
6. **[iss-195](../../work/issues/open/iss-195-scanner-openended-heuristic-cost-regression-on-network-patterns.md)**
   (minor) — the rigid/open-ended heuristic sends every IPv4/IPv6 match through
   the backward search. Fix-eligible by the 2026-08-08 ruling (it escaped the
   adjacency shelving: a cost bug, not a window-truncation bug).
   Autonomous-eligible.
7. **[iss-147](../../work/issues/resolved/iss-147-guard-load-reads-abcd-guard-json-from-the-working-tree-so-a.md)**
   (minor) — working-tree guard config is an instant disarm.
8. **[iss-148](../../work/issues/resolved/iss-148-guard-registry-coverage-gaps-found-while-wiring-itd-103-regi.md)**
   (minor) — registry coverage gaps; every entry lands fixture-first per the
   v0.5.0 plan's rule.
9. **[iss-174](../../work/issues/open/iss-174-rules-override-withholds-bundled-default-upgrades.md)**
   (minor) — a repo's rules override silently withholds bundled security
   upgrades.

## Workstream D — the install path tells the truth (UX bugs)

No ordering constraints among them; each is small and autonomous-eligible
unless its body says otherwise:
[iss-33](../../work/issues/open/iss-33-ahoy-verb-hygiene.md) (unvalidated
interactive answers persisted),
[iss-221](../../work/issues/open/iss-221-refounding-lineage-prompt-is-a-one-shot.md),
[iss-222](../../work/issues/resolved/iss-222-install-dev-silent-noop-over-unowned-wrapper.md),
[iss-227](../../work/issues/open/iss-227-installdevshim-silently-swallows-failures-the-os-remove-mkdi.md),
[iss-228](../../work/issues/open/iss-228-the-plugin-root-binary-repo-root-abcd-bin-abcd-darwin-arm64.md).

## Structural tier — designed, deliberately not next

**[iss-229](../../work/issues/resolved/iss-229-scanner-adjacency-galloping-probe-structural-fix.md)**
— the galloping-probe (`trueMatchEnd`) replacement for the fixed adjacency
window, superseding
[iss-189](../../work/issues/resolved/iss-189-adjacency-probe-window-edge-false-positive.md)
and
[iss-190](../../work/issues/resolved/iss-190-scanner-adjacency-recovery-is-capped-by-maxadjacencyprobewin.md)
(their repro shapes are its acceptance corpus). Human-paired or a deliberate
root-cause escalation round, per the 2026-08-08 shelving: local patches to the
window-edge discard logic are exhausted — three independent BLOCKs — so the
only admissible attempt is the structural design itself.

## Ordering and collisions

- C1 (iss-200) and C7/C8 (iss-147/148) all touch the guard match/registry
  seam; land C1 first — it is the live bypass — then the two hardening items.
- C2 and C3 are one seam (scanner construction and its audit consumer); the
  pair is atomic in ordering even if they land as two PRs.
- C6 (iss-195) and the structural tier (iss-229) both touch
  `internal/adapter/scanner/scanner.go`'s adjacency path. iss-195 lands first
  — it is small and independent — and iss-229's later rewrite of the probe
  machinery must keep iss-195's regression test green, not re-litigate it.
- Workstream A (Cut B) and C items are independent; A's own collision notes
  (release.yml vs iss-209's parity test) live in the install-experience plan.

## STOP conditions (this plan)

1. **The shelving holds.** Any fix attempt that patches the window-edge
   discard/skip logic locally — rather than implementing iss-229's structural
   design — is a STOP, whoever proposes it.
2. **First-attempt BLOCK on a fix** stops that item cleanly (the iss-200
   precedent): report, leave the branch unpushed, move on.
3. **Security-review BLOCK** stops the change (standing rule).
4. **Missing or ambiguous record** — an issue body that does not support the
   work as scoped: fail closed, never synthesise a substitute.
5. **No release-gate or receipt-contract edits** ride any item here.

## Explicitly out

Superseded, not deferred: iss-189/190 (resolved as superseded by iss-229).
Deferred to their own plans: everything the facilitator touches
(facilitator-experience) and everything maintainer-loop-only
(predictable-development, including iss-219 — real, but an installer never
runs the dev-install test suite).
