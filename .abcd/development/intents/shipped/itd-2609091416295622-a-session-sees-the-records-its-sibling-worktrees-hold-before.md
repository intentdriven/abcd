---
id: itd-2609091416295622
slug: a-session-sees-the-records-its-sibling-worktrees-hold-before
spec_id: spc-2609202056480020
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609091034175565]
severity: major
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-2609091416304128, itd-2609091014076309, itd-148, itd-2609150819440345, itd-2609201916151817]
impact: additive
---

# A session sees the records its peers hold before it mints or fixes one

## Press Release

> **Before a session captures, fixes or files anything, it can see what every peer already holds.** One read-only reader answers the question from two sources today, the sibling worktrees' disks and the local branches, and from the register later. It reaches a session three ways: `capture resolve`, the record dispatcher and `intent audit` name the peer that holds a record when they cannot find it here, instead of answering not found; the status board carries one line whenever a peer holds something this tree does not; and one read-only command prints the whole picture on demand. It shows issues and intent drafts alike, skips peers whose branch is merged or whose worktree is gone, and says out loud when a peer's ledger is in a state it will not read. Nothing is written: no claim, no lease, no session key. The implement verb reads the same picture before it picks a record.
>
> "I had two sessions running in sibling worktrees, and one of them was about to capture an issue the other had captured an hour earlier, unpushed," said Maya, an autonomous-development practitioner who runs several agent sessions against one record. "The file was sitting on the disk the whole time, two directories over. Now the resolve tells me who has it, before I have done anything."

## Why This Matters

Every collision on record so far happened between sessions that could have read each other's files. On 2026-09-01 a peer session re-fixed two issues that a paused branch had already fixed and not pushed. On 2026-09-09 two sessions held worktrees off one checkout, nothing pushed: one nearly minted a record its peer had already captured, and one nearly created a worktree over a peer's. On 2026-09-18, in a managed repository, a lane's `capture resolve` answered not found for a record that existed one worktree over, an audit agent working from a stale worktree reported three shipped intents as existing nowhere, and lanes merged each other's branches to resolve records. All of it is in [iss-2609020716570699](../../../work/issues/open/iss-2609020716570699-nothing-tells-an-agent-that-a-record-it-is-about-to-fix-has.md) and its corroboration. In each case the record existed on the machine, and nothing rendered it; the convention in `AGENTS.md` that asks a session to scan for peers points at a surface abcd never renders.

The claim record this was split from proposed a machine-scoped lease, a `claimed_by` stamp, refusals in the write verbs and a pushed duplicate guard. Two adversarial reviews on 2026-09-09 (their application is the DECISIONS.md ruling of that date) found the expensive parts unsound for a human session and found that the collisions on record needed none of it. The maintainer ruled a split into the listing (this record), the upstream-terminal refusal and the claim. On 2026-09-20 the product thinker settled the four records' shape (below): this listing is the read side, the claim the write side, the register where both live across machines, and the implement verb the consumer.

## Decisions

Settled with the product thinker on 2026-09-20:

1. **Four records, linked.** This listing (read), the claim `itd-2609091034175565` (write), the register `itd-2609150819440345` (where claims and holdings live, across accounts and across machines over a network such as a tailnet), and the implement verb `itd-2609201916151817` (claims, checks, implements or drops and picks the next). Each is planned on its own; this one first.
2. **The primitive: both sources through one reader.** The sibling worktrees' disks (sees uncommitted captures) and the local branches through the common git dir (sees a branch checked out nowhere), behind one reader the upstream-terminal refusal and, later, the register share.
3. **The home: the refusals, the board, and a standalone read command.** `capture resolve`, `abcd <record-id>` and `intent audit` name the peer holding a record they cannot find here; the board carries one line; one read-only command (working name `peers`; the name is settled at build time) prints the whole picture.
4. **Spent peers are skipped.** A worktree whose directory is gone, or whose branch is merged into the default branch, is left out of the diff and named in a count.
5. **A peer whose ledger holds one id in two status folders is marked, not read**, and the other peers render normally; the block names the id and the remedy.
6. **Issues and intent drafts** are both listed, by the same filename rule.
7. **Relation to `itd-148`/`spc-42`:** this is a distinct read primitive that the worktree render consumes; it does not ship that render early.
8. The always-on provenance line on every ledger verb is its own record, `iss-2609202053570475`.

## What's In Scope

- **One reader, two sources.** Given this checkout, the reader enumerates the peers git names (`git worktree list --porcelain`, each candidate confirmed from its own `--git-common-dir`; every local branch through the common dir) and, per peer, reads the ids under `.abcd/work/issues/{open,resolved,wontfix}/` and `.abcd/development/intents/drafts/` by filename, opening a record only for its title through the same guarded reader every record read uses.
- **The diff.** Per live peer: Records open there and absent here; records open here and terminal there; drafts there and absent here. Each id with its title where the file is readable, the id alone where it is not.
- **The not-found paths.** `capture resolve`, `abcd <record-id>` and `intent audit`, when the record is not in this tree and a peer holds it, refuse naming the peer's branch and path and the folder that holds it.
- **The board line.** The count of live peers and the count of ids across the diff, present only when the diff is non-empty.
- **The standalone command**, text and `--json`, home paths redacted on every stream.
- **`AGENTS.md`'s concurrent-sessions convention names the command** as the scan-before-mutating step.

## What's Out of Scope

- Any write: No claim, no lease, no stamp, no hook, no session key (the claim record's).
- Presence or liveness: A peer holding a record says the record exists, not that a session does.
- Peers beyond this checkout's common dir: A second clone, another account, another machine (the register's).
- A record in a stash or an unsaved buffer.
- A record already terminal on the default branch (the upstream-terminal refusal's).
- Judging that two differently worded records describe one observation (`itd-87`'s).
- Creating, moving or reclaiming worktrees (the worktree store's and `itd-148`'s).

## Mechanism

We expect a read-only view of what peers hold to stop the local collisions on record because each was a failure of visibility and not of will: The peer's record existed on the machine before the collision, and the verb that collided answered not found rather than naming it. It is falsified if a session that was shown a peer's holding, in a refusal, on the board or in the command's output, still minted a duplicate of it or re-fixed a record shown as terminal elsewhere; the render's own output and the two records' timestamp ids tell that case apart from a session that never looked. It is also falsified, differently, if the next collision on record comes from a peer this reader cannot see, a second clone or another machine, which is the register's case.

## Scope Conditions

- Holds for the peers that share this checkout's common git dir: Its worktrees and its local branches. A second clone, another account and another machine are the register's boundary. <!-- cond: cond-2609202056488592 -->
- Holds for a record on a peer's disk or in a peer's branch; a stash or an unsaved buffer is invisible. <!-- cond: cond-2609202056489916 -->
- Holds at the scale of this checkout, some thirty worktrees and a few hundred branches, each ledger some hundreds of records; behaviour past that, or on a network filesystem, is unmeasured. <!-- cond: cond-2609202056486227 -->
- Holds where git answers for the peer under abcd's isolated environment; a peer git refuses (another uid's checkout, unless `~/.abcd/trusted-roots` re-admits it) is named and not read. <!-- cond: cond-2609202056485097 -->
- macOS and Linux; the porcelain listing's path form on Windows is untested. <!-- cond: cond-2609202056483851 -->
- Assumes the peer's ledger uses the committed layout, so a peer at a commit before that layout existed contributes no rows and says so. <!-- cond: cond-2609202056489557 -->

## Acceptance Criteria

- **Given** a record captured under a sibling worktree's `open/` and absent from every status folder here, **when** the standalone command runs, **then** one row under that peer names the id and its title, the block names the peer's branch and home-redacted path, the exit code is 0, and `git status --porcelain` in both trees is byte-identical to before.
- **Given** a record open here and present under a peer's `resolved/` or `wontfix/`, **when** the command runs, **then** the record appears under that peer naming which terminal folder holds it.
- **Given** a record open here and resolved on a local branch that no worktree has checked out, **when** the command runs, **then** that branch appears as a peer and the record under it.
- **Given** an intent draft under a peer's `drafts/` with no draft of that id here, **when** the command runs, **then** it appears under that peer as a draft.
- **Given** `capture resolve <iss-N>` for a record not in this tree that a peer holds, **when** it runs, **then** the refusal names the peer's branch, path and folder instead of not found; the same for `abcd <iss-N>` and `intent audit <itd-N>`.
- **Given** a peer whose branch is merged into the default branch, or whose worktree directory is gone, **when** the command runs, **then** it contributes no rows and is counted in a skipped-peers line.
- **Given** a peer whose ledger holds one id in two status folders, **when** the command runs, **then** that peer's block says it was not read, names the id and the remedy, and every other peer renders normally.
- **Given** a directory `git worktree list` names whose own common dir is not this checkout's, or for which git refuses to answer, **when** the command runs, **then** it is not read and its block says why.
- **Given** a checkout with no peers, **when** the command runs, **then** it reports none, exits 0, and the board carries no line.
- **Given** a non-empty diff, **when** the board renders, **then** one line states the live-peer count and the id count; **given** an empty diff, the line is absent.
- **Given** `--json`, **when** the command runs, **then** the payload carries the same blocks, and no value carries an unredacted home path.
- **Given** a peer's record file that is unreadable or malformed, **when** the command runs, **then** the id appears without a title and the command exits 0.
- **Given** `AGENTS.md`'s concurrent-sessions convention, **when** it is read, **then** the scan-before-mutating step names the command beside the harness's session listing.

## Typed Links

- **builds_on `itd-2609091034175565`** (the claim): The record this was split from on 2026-09-09; the write side this read side answers.
- **refines `itd-2609091416304128`** (the upstream-terminal refusal): Shares the reader; it judges against the fetched default branch, this against local peers.
- **refines `itd-2609091014076309`** (the worktree store) and **`itd-148`** (every change in its own worktree): Their renders list worktrees; this reader gives them the ledger columns and does not ship their render.
- **built on by `itd-2609150819440345`** (the register): Adds the network source to this reader.
- **built on by `itd-2609201916151817`** (the implement verb): Reads this picture before it picks a record.
- Prose cross-references, because no schema field carries the relation (`iss-2609091256264547`): `iss-2609020716570699` (the collisions), `iss-2608220750029993` (the presence lease this is not).

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-a5f2f1052770 -->
Fidelity review — receipt rcp-a5f2f1052770 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:79c4c254f138aa0015a9eb00e07f5d1cfbb656bf88a89265f57fde8c478d4c82
Input attestations: diff:8486c141..cf1247d4 (PR #666, merged 20cc37eb; tree read at cede78b8)@-;

Acceptance rollup: MET 13 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: the core test captures under a sibling worktree, reads, and asserts the row plus byte-identical porcelain in both trees; the surface test asserts the branch, the home-redacted path, the id and title in text and --json
  evidence: internal/core/peers/peers_test.go:122 — "func TestACaptureOnASiblingDiskIsARowUnderThatPeer"
  evidence: internal/core/peers/peers_test.go:129 — "if porcelain(f, f.here()) != beforeHere || porcelain(f, a) != beforeA {"
  evidence: internal/surface/cli/peers_surface_test.go:91 — "for _, want := range []string{"feat/a", "~/wt/a", "iss-100", "open there, absent here", "A finding the peer captured"} {"
- ac-2 — MET: the diff computes an open-here/terminal-there row carrying the peer's folder, and the test names the folder
  evidence: internal/core/peers/peers.go:62 — "KindTerminalThere Kind = "terminal-there""
  evidence: internal/core/peers/peers.go:77 — "Folder string `json:"folder"`"
  evidence: internal/core/peers/peers_test.go:154 — "func TestARecordOpenHereAndTerminalThereNamesTheFolder"
- ac-3 — MET: every local branch no worktree has checked out is read from the common dir with ls-tree as a branch peer, test-held
  evidence: internal/core/peers/read.go:142 — "rep.Peers = append(rep.Peers, readBranch(root, b))"
  evidence: internal/core/peers/read.go:430 — "args := []string{"ls-tree", "-r", "-z", "--full-tree", ref, "--"}"
  evidence: internal/core/peers/peers_test.go:182 — "func TestABranchCheckedOutNowhereIsAPeer"
- ac-4 — MET: a draft in a peer's drafts/ absent here is a draft-there row, test-held
  evidence: internal/core/peers/peers.go:67 — "KindDraftThere Kind = "draft-there""
  evidence: internal/core/peers/peers_test.go:204 — "func TestAPeerDraftAbsentHereIsADraftRow"
- ac-5 — MET: all three verbs route their not-found error through peerHeldRefusal, which names each holder's branch, redacted path and folder, and the surface test runs all three and asserts 'not found' is gone
  evidence: internal/surface/cli/cli.go:221 — "return peerHeldRefusal(cwd, "", args[0], err)"
  evidence: internal/surface/cli/cli.go:2247 — "return peerHeldRefusal(repoRoot, "abcd intent audit: ", args[0],"
  evidence: internal/surface/cli/cli.go:3409 — "return peerHeldRefusal(repoRoot, "abcd capture resolve: ", args[0], err)"
  evidence: internal/surface/cli/peers.go:236 — "holders = append(holders, h+" holds it in "+l.Folder+"/")"
  evidence: internal/surface/cli/peers_surface_test.go:167 — "func TestNotFoundPathsNameThePeerThatHoldsTheRecord"
- ac-6 — MET: a gone directory and a branch merged by ancestry (with clean record folders) are returned as Skipped with a reason, rendered in a skipped count on the header line, and test-held; a merged worktree with uncommitted records deliberately stays live
  evidence: internal/core/peers/read.go:158 — "return p, &Skipped{Source: SourceWorktree, Branch: wt.branch, Path: wt.path, Reason: SkipGone}, true"
  evidence: internal/core/peers/read.go:186 — "return p, &Skipped{Source: SourceWorktree, Branch: wt.branch, Path: wt.path, Reason: SkipMerged}, false"
  evidence: internal/surface/cli/peers.go:157 — "return fmt.Sprintf("; %d skipped (%s)", len(sk), strings.Join(parts, ", "))"
  evidence: internal/core/peers/peers_test.go:225 — "func TestSpentPeersAreSkippedAndCounted"
- ac-7 — MET: judgeHoldings marks a peer holding one id in two folders with the ids and the remedy and no rows, and the test asserts the healthy sibling still renders
  evidence: internal/core/peers/read.go:240 — "return "its ledger holds one id in two places (" + strings.Join(split, "; ") +"
  evidence: internal/core/peers/peers_test.go:300 — "func TestASplitLedgerPeerIsMarkedAndOthersRender"
- ac-8 — MET: the candidate's own --git-common-dir is compared to this checkout's, and a refusal or a mismatch sets NotRead with the reason, test-held for both
  evidence: internal/core/peers/read.go:168 — "p.NotRead = "git refused to answer for it: " + firstLine(err.Error())"
  evidence: internal/core/peers/read.go:172 — "p.NotRead = "its own common dir is not this checkout's, so it belongs to another repository""
  evidence: internal/core/peers/peers_test.go:329 — "func TestAForeignOrRefusedWorktreeIsNamedNotRead"
- ac-9 — MET: with no peers the command prints 'no peers' and the board carries no line in text or --json, test-held on the surface and the core
  evidence: internal/surface/cli/peers.go:96 — "fmt.Fprintln(w, "abcd peers — no peers")"
  evidence: internal/surface/cli/peers_surface_test.go:135 — "if out := string(runCLI(t, "peers")); !strings.Contains(out, "no peers") {"
  evidence: internal/core/peers/peers_test.go:356 — "func TestACheckoutWithNoPeersReportsNone"
- ac-10 — MET: the board's peers line is built only when the id count is non-zero and carries the live-peer and id counts, asserted present with one peer and absent with none
  evidence: internal/surface/cli/peers.go:187 — "if rep.IDCount() == 0 {"
  evidence: internal/surface/cli/peers.go:190 — "return &boardPeersLine{Live: rep.Live(), IDs: rep.IDCount()}"
  evidence: internal/surface/cli/peers_surface_test.go:132 — "func TestTheBoardCarriesAPeersLineOnlyWhenPeersHoldSomething"
- ac-11 — MET: the --json payload is built from the same Report with every peer and skipped path and NotRead reason passed through RedactHome, and the surface test asserts no home path in the raw JSON
  evidence: internal/surface/cli/peers.go:80 — "p.Path = fsutil.RedactHome(p.Path)"
  evidence: internal/surface/cli/peers.go:85 — "s.Path = fsutil.RedactHome(s.Path)"
  evidence: internal/surface/cli/peers_surface_test.go:99 — "noHomePath(t, home, string(raw))"
- ac-12 — MET: a title read that fails returns the empty string and the id is listed anyway; the test covers a malformed and a mode-0 file with the read succeeding
  evidence: internal/core/peers/read.go:321 — "if err != nil {"
  evidence: internal/core/peers/peers.go:79 — "malformed; the id is listed either way."
  evidence: internal/core/peers/peers_test.go:369 — "func TestAnUnreadableOrMalformedRecordListsTheIDAlone"
- ac-13 — MET: the scan-before-mutating step in AGENTS.md names `abcd peers` beside the harness's session listing, and a test reads that step and asserts both
  evidence: AGENTS.md:209 — "also run `go run ./cmd/abcd peers` (`--json` for a machine reader): it lists"
  evidence: internal/surface/cli/peers_surface_test.go:262 — "func TestTheScanBeforeMutatingConventionNamesThePeerListing"

Gap audit:
- honoured:
  - one reader, two sources, nothing written
    evidence: internal/core/peers/peers.go:53 — "var Sources = []Source{SourceWorktree, SourceBranch}"
    evidence: internal/core/peers/peers.go:9 — "It opens no file for writing, runs only read-only git commands"
  - the three not-found paths name the peer
    evidence: internal/surface/cli/peers_surface_test.go:167 — "func TestNotFoundPathsNameThePeerThatHoldsTheRecord"
  - the register source is a declared empty slot, and a Report says so
    evidence: internal/core/peers/peers.go:45 — "SourceRegister is the slot the register intent (itd-2609150819440345)"
  - a peer's branch and path are sanitised before they reach a stream
    evidence: internal/surface/cli/peers_surface_test.go:221 — "func TestThePeerHeldRefusalSanitisesThePeersBranchAndPath"
  - a gone or refused worktree's unmerged branch is still read from the store
    evidence: internal/core/peers/peers_test.go:260 — "func TestAGoneOrRefusedWorktreesUnmergedBranchIsReadFromTheStore"
- diverged:
  - the not-found paths look up drafts/ only: Locate spans every intent bucket, so intent audit names a peer's shipped/ copy too (a widening, not a loss)
    evidence: internal/surface/cli/peers_surface_test.go:184 — "{"intent audit", []string{"intent", "audit", "itd-77"}, []string{"side", "shipped/"}},"
- missing: (none)

Scope-condition dispositions:
- cond-2609202056488592 — survived: a listed directory whose own common dir is not this checkout's is named and not read, and the sources are the two local ones with the register slot empty
  evidence: internal/core/peers/read.go:171 — "if realPath(theirs) != realPath(common) {"
  evidence: internal/core/peers/peers.go:53 — "var Sources = []Source{SourceWorktree, SourceBranch}"
- cond-2609202056489916 — survived: holdings are read off a worktree's disk or from a branch's tree with ls-tree; nothing reads a stash or a buffer
  evidence: internal/core/peers/read.go:378 — "func scanDisk(root string) (holdings, bool, error) {"
  evidence: internal/core/peers/read.go:429 — "func scanTree(root, ref string) (holdings, bool, error) {"
- cond-2609202056486227 — untested: no test or artefact in the delivered diff exercises the reader at thirty worktrees or a few hundred branches, nor on a network filesystem
- cond-2609202056485097 — survived: a worktree git refuses to answer for is named with the refusal and not read, test-held with a dangling gitdir pointer
  evidence: internal/core/peers/read.go:166 — "theirs, err := commonDir(wt.path)"
  evidence: internal/core/peers/peers_test.go:329 — "func TestAForeignOrRefusedWorktreeIsNamedNotRead"
- cond-2609202056483851 — survived: the tests run on the ubuntu and macos legs of the CI matrix and the porcelain parser is tested for both path forms git emits; Windows is not in the matrix, as the condition states
  evidence: .github/workflows/ci.yml:230 — "os: [ubuntu-latest, macos-latest]"
  evidence: internal/gitutil/worktree_test.go:14 — "func TestParseWorktreeListReadsBothPorcelainForms"
- cond-2609202056489557 — survived: a peer holding no records at the committed layout is marked NotRead with that reason and contributes no rows
  evidence: internal/core/peers/read.go:224 — "return "it holds no records at the committed layout (" + capture.LedgerRelPath + "/, " +"
## Grounds

- pursued: the pilot and the big run put four to seven lanes on one checkout this week, and every collision on record was a failure of visibility, not of will; we expect a read-only view of what peers hold, delivered in the verbs' own refusals, to stop them; shown wrong if a lane that was shown a peer's holding still duplicates or re-fixes it, or if the next collision comes from a peer this reader cannot see
