---
id: itd-2609091718566731
slug: transcripts-already-on-disk-are-recovered-into-the-right-rep
spec_id: spc-2609091722230648
kind: standalone
suggested_kind: null
reclassification_history: []
related_adrs: [adr-29]
builds_on: [itd-2609090559376002]
severity: major
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Transcripts Already on Disk Are Recovered into the Right Repository's Corpus, Under That Repository's Own Redaction

## Press Release

> **abcd recovers the transcripts it never captured, into the repository that owns them, under that repository's own redaction rules.** Capturing sub-agents from now on leaves everything before now on the floor: months of delegated work sitting in the harness's own directories, ageing out under a retention sweep abcd does not control. `abcd history ingest --into <repo>` brings that material into the corpus. The destination repository is an operand with no default, so one repository's scanner configuration governs its own transcripts and can never be applied to another's; a transcript whose owner cannot be established is skipped and named rather than filed by guess; and a transcript whose repository is not on this machine at all is an orphan — ignored, reported, and stored only when a repository claims its project by name. The same run repairs the records already in the store that were filed under a hand-made composite identifier, which neither the reader nor the store could resolve.
>
> "The corpus was going to start on the day I fixed it, and everything I had actually done was going to be the part that was missing," said Maya, an autonomous-development practitioner. "What I could not accept was a recovery tool that guessed. If it cannot tell which repository a transcript belongs to, I want it to say so and leave it alone — a transcript redacted by the wrong repository's rules is worse than a transcript I never recovered."

## Why This Matters

`itd-2609090559376002` closes the capture gap prospectively. It does nothing about the past, and the past is where the evidence is: on the machine this was built on, 968 sub-agent transcript files holding 673 MB were on disk and had never been read by abcd, against 68 session files and 206 MB that had. The harness's retention sweep eventually removes them, so "we will get to it" resolves to "we lost it".

There is a second, smaller population with the same shape. Before the record carried lineage fields, the workaround in use was to invent a composite session identifier by concatenating a truncated parent id with the agent id. 176 of the store's 267 records carried one. That identifier fails in both directions — a reader holding the full session id the harness gives them cannot find the record, and a reader holding the record cannot recover the full parent id, because the truncation is lossy — so those records were in the store and out of reach. They need repairing, not re-ingesting: the store holds the only copy.

Both are recovery, and recovery is where the redaction question becomes sharp. Live capture takes its repository from the working directory it runs in, and that is correct because the session ran there. An operator recovering a backlog is standing wherever they happen to be standing, which has nothing to do with where the transcripts came from. Deriving the destination from the working directory would silently redact one repository's transcripts under another repository's `pii.json` and `gitleaks.json` — a privacy fault, not a misfiling. So the destination is asked for, every time. `--into .` is a fine answer; an unasked question is not.

The recovery run this was built against stored 869 transcripts for this repository and 197 for a second one on the same machine, repaired all 176 composite records, and left the store holding 1104 records. Thirteen transcripts were refused outright by the fail-closed scanner over network addresses it could not redact — refused, reported, and not stored, which is the behaviour the corpus is supposed to have.

## Typed Links

- **builds_on `itd-2609090559376002`** (sub-agent transcript capture): consumes the lineage fields and the fail-closed `Capture` path that intent put on the record. Without them an ingested sub-agent has nowhere to record which session and which agent produced it.
- **refines `adr-29`** (native transcript corpus): a second door into the store that ADR established, under the same per-repo keying and the same redact-on-write discipline. It adds no third write path.

## What's In Scope

- **Ingest of history that already exists:** transcripts still on disk but never captured are brought into the corpus for a repository the caller names, under that repository's own redaction configuration and never another's.
- **A destination that is asked for, not derived.** The destination repository root is a required operand with no default, and the run says which repository it wrote into.
- **Explicit sources.** The paths to read are given on the command line or declared in the repository's own configuration; no vendor directory is ever assumed, so nothing here depends on the harness's on-disk layout.
- **An owner that is established or refused, never guessed.** A transcript owned by another repository is skipped and its owner named by root SHA; one that names two repositories is skipped rather than split; one that names no repository this machine has is an orphan.
- **Orphans are ignored, reported, and adopted only by name**, per repository, and an adopted record carries the project it was adopted from so the adoption is a property of the artefact rather than of a run's output.
- **Idempotence:** ingesting the same material twice adds nothing.
- **Repair of the records already in the store** that were filed under the pre-lineage composite identifier, recovering the full parent session from the record's own body and reporting by default, writing only when told to.

## What's Out of Scope

- **Capturing new sub-agents.** The live path is `itd-2609090559376002`.
- **Reconstruction and telemetry.** Reading a recovered session back as one artefact is `itd-2609091718595846`.
- **Interactive questions in the core.** Core returns the orphan list and ingests none of them whatever the policy says; asking a human is a transport concern and lives at the front door.
- **Redesigning the corpus.** The store's per-repo keying, its provisioning, and its two-stage redaction stay as they are.
- **The harness's own retention policy**, and any attempt to slow it down.
- **Renaming or re-hashing repaired records.** A repaired record keeps its filename and its source digest, so every path a reader already holds still resolves and the record still dedups against a re-capture of the same bytes.

## Mechanism

We expect making the destination repository an explicit, defaultless operand to be what keeps a recovery run honest, because the scanner is constructed from the destination's repository root and from nothing else, so the seam that decides redaction policy is the same seam the operator had to answer — there is no path by which the working directory can supply it. We expect the working directory recorded inside a transcript's own lines to be the only sound owner signal, resolved per session before per file, because the harness's project directory name is not reversible to a filesystem path and a sub-agent handed a worktree that the harness has since deleted has no surviving directory of its own. This is shown wrong if a transcript's recorded working directories do not resolve often enough for the recovery to be worth running, if sessions routinely record two repositories so that the ambiguity refusal swallows the corpus, or if the per-file read cost makes a backlog-sized run impractical.

## Scope Conditions

- Holds where the transcripts on disk are line-delimited JSON of the same shape the session transcript uses, one transcript per file, each naming exactly one session. <!-- cond: cond-2609091722236462 -->
- Assumes a transcript's owning repository is determined by the working directory recorded inside its own lines, resolved per session before per file, and never by decoding the harness's project directory name, which is not reversible to a path. <!-- cond: cond-2609091722231767 -->
- Assumes a session run in a worktree belongs to the store of the repository that worktree derives from, and that the worktree may no longer exist when the recovery runs. <!-- cond: cond-2609091722232975 -->
- The destination repository's redaction configuration is authoritative for everything stored into it. A transcript the destination's fail-closed scanner refuses is not stored anywhere, and the refusal is reported rather than worked around. <!-- cond: cond-2609091722237552 -->
- Sources are explicit paths, or roots the destination repository declares for itself. Nothing here reads a vendor directory by convention, so a harness that moves its files changes what an operator types and nothing else. <!-- cond: cond-2609091722234418 -->
- Holds at backlog scale — roughly a thousand transcripts and a gigabyte in one run — with each file read once and one file resident at a time. <!-- cond: cond-2609091722233051 -->

## Acceptance Criteria

- **Given** transcripts on disk that were never captured, **when** an operator ingests them for a named destination repository, **then** they are redacted under that repository's own configuration and stored in that repository's corpus, and ingesting the same material twice adds nothing.
- **Given** a transcript whose owning repository cannot be identified, **when** ingest runs without configuration naming a destination for it, **then** it is skipped and reported rather than filed anywhere by guess.
- **Given** a repository configured to adopt a named orphaned project, **when** ingest runs, **then** that project's transcripts are stored in that repository's corpus and the adoption is recorded on the record itself.

## Open Questions

- **Whether a scheduled recovery should exist at all.** Today ingest is a verb an operator runs. A repository that ingests regularly declares its roots and retypes nothing, but nothing runs it on a timer, and the harness's retention sweep does. Whether that gap is worth closing is not settled here.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-813aa9d78674 -->
Fidelity review — receipt rcp-813aa9d78674 (verifier abcd:intent-auditor claude-opus-5[1m]).

Provenance: abcd:intent-auditor@claude-opus-5[1m] · rubric_hash sha256:542ed2cd51ff938717a3f47b2b332e8d47910beec0ca7ecdfd238ae7edf5ced5 · prompt_hash sha256:de0e9e0fc152462b89e15c77658fa94f091fc157a3557537344f3584254310a6
Input attestations: diff:319da670 feat: history migrate repairs the composite records, history ingest takes a destination (branch feat/sub-agent-transcript-capture)@-;

Acceptance rollup: MET 2 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: Every reachable path holds: the destination is refused when absent, the scanner is built from dest.RepoRoot alone, cwd is never consulted by ingest, and a second run writes nothing; the named caveat is that Ingest validates RootSHA's shape but never that RepoRoot and RootSHA describe the same repository, so the core seam that carries the whole privacy argument would accept a self-inconsistent Destination.
  evidence: internal/core/history/history.go:212 — "sc, err := scanner.New(repoRoot)"
  evidence: internal/core/history/ingest.go:167 — "ingest needs an explicit destination repository root; it is never derived from the working directory"
  evidence: internal/core/history/ingest.go:168 — "if !rootSHARe.MatchString(dest.RootSHA)"
  evidence: internal/core/history/history.go:205 — "if r.SourceSHA256 == sourceSHA && r.SessionID == sessionID &&"
  evidence: internal/surface/cli/history_recovery_test.go:186 — "TestHistoryIngestWritesIntoTheNamedRepositoryNotTheWorkingDirectory"
  evidence: internal/core/history/ingest_test.go:227 — "TestIngestIsIdempotent"
- ac-2 — MET: Both unidentifiable populations are reported and neither is stored: an ambiguous owner becomes a skip carrying its reason, and an unresolved owner with no adoption naming it becomes a reported orphan and returns before any write.
  evidence: internal/core/history/ingest.go:322 — "Reason: SkipAmbiguousOwner"
  evidence: internal/core/history/ingest.go:326 — "if _, claimed := adopt[p.project]; !claimed {"
  evidence: internal/core/history/ingest.go:530 — "if p.sessionIDs != 1 || p.agentIDs > 1 {"
  evidence: internal/core/history/ingest_test.go:164 — "the store must be untouched by an ignored orphan"
  evidence: internal/core/history/ingest_test.go:259 — "an ambiguously owned transcript must not be stored"
- ac-3 — MET: Adoption is reachable only when the owner is unresolved and unambiguous and the project is named by the destination's own config or by --adopt, and the stored record carries adopted_project in its frontmatter, asserted on disk.
  evidence: internal/core/history/ingest.go:333 — "store(dest, opts, p, "adopted", p.project, res)"
  evidence: internal/core/history/store.go:270 — "{fmAdoptedProject, r.AdoptedProject},"
  evidence: internal/core/history/ingest_test.go:220 — "an adopted record must carry "adopted_project: claimed""
  evidence: internal/surface/cli/history_recovery.go:157 — "cfg, err := history.LoadConfig(dest.RepoRoot)"

Gap audit:
- honoured:
  - The destination repository is an operand with no default, so one repository's scanner configuration governs its own transcripts and can never be applied to another's.
    evidence: internal/surface/cli/history_recovery.go:220 — "--into < repo-root> is required and has no default"
    evidence: internal/core/history/history.go:212 — "sc, err := scanner.New(repoRoot)"
  - Ownership is resolved for the session before the file, so a sub-agent whose worktree the harness deleted is not orphaned when its session resolves.
    evidence: internal/core/history/ingest.go:226 — "func placeSessions(probes []transcriptProbe) map[string]sessionPlacement"
    evidence: internal/core/history/ingest.go:311 — "Fallback: the file's own recorded directory."
    evidence: internal/core/history/ingest_test.go:104 — "TestIngestPlacesTheSessionBeforeTheFile"
  - An orphan is ignored, reported, and stored only when a repository claims its project by name, with the adoption a property of the artefact.
    evidence: internal/core/history/ingest.go:328 — "res.Orphans = append(res.Orphans, Orphan{"
    evidence: internal/surface/cli/history_recovery.go:295 — "orphan %s under project %s (recorded cwd %s) - ignored"
  - The same run repairs the records filed under the hand-made composite identifier, recovering the full parent session from the record's own body and writing only under --apply.
    evidence: internal/core/history/migrate.go:137 — "func Migrate(rootSHA string, opts MigrateOptions) (MigrateResult, error)"
    evidence: internal/core/history/migrate.go:258 — "if !opts.Apply {"
    evidence: internal/core/history/migrate.go:198 — "full, err := recoverSessionID(body, prefix)"
  - A transcript the destination's fail-closed scanner refuses is not stored and the refusal is reported.
    evidence: internal/core/history/ingest.go:377 — "res.Failed = append(res.Failed, IngestFailure{Path: p.path, Err: err.Error()})"
    evidence: internal/core/history/history.go:271 — "residual := scanner.BlockingResidual(sc.ScanText(redacted, "transcript"))"
- diverged:
  - The spec describes the ac-1 redaction test as two fixture repositories with different rules, asserting the destination's applied and the source's not; the delivered test uses ONE repository and asserts only that the destination's own detector fired.
    evidence: internal/core/history/ingest_test.go:280 — "TestIngestRedactsUnderTheDestinationsOwnConfiguration"
    evidence: internal/core/history/ingest_test.go:315 — "the destination's own detector did not govern its own store"
  - The spec gives the composite one shape, < prefix>--agent-< id>; the delivery splits from the right to also read a workflow-shaped composite, a knowing divergence the commit message records.
    evidence: internal/core/history/migrate.go:284 — "A left-to-right split would read a workflow id as part of the prefix"
  - SessionOwner is exported, has zero callers, and duplicates the session-placement logic inline - the exported-with-no-front-door state the repository's own wired-or-it-isn't-done boundary forbids.
    evidence: internal/core/history/ingest.go:545 — "func SessionOwner(sessionID string) (string, error)"
    evidence: internal/core/history/ingest.go:267 — "if sha := storeOwner(p.sessionID); sha != "" {"
- missing:
  - The press release and the In-Scope bullets promise the composite-record repair, but the intent's Acceptance Criteria judge only ingest, so the second promised behaviour has no acceptance bar in the record at all - a gap in the record, not in the code.
    evidence: .abcd/development/intents/shipped/itd-2609091718566731-transcripts-already-on-disk-are-recovered-into-the-right-rep.md:47 — "Repair of the records already in the store"
    evidence: .abcd/development/intents/shipped/itd-2609091718566731-transcripts-already-on-disk-are-recovered-into-the-right-rep.md:73 — "the three Acceptance Criteria bullets name only ingest"
  - The two file-integrity skip reasons, no-session-id and ambiguous-agent-id, have no test asserting the reason string an operator reads - the spec's own Uncertainties says so.
    evidence: internal/core/history/ingest.go:65 — "SkipNoSession = "no-session-id""

Scope-condition dispositions:
- cond-2609091722236462 — survived: The probe reads line-delimited JSON and enforces exactly one session per file, refusing anything else rather than splitting it, so the assumed shape held and non-conforming files were reported.
  evidence: internal/core/history/ingest.go:530 — "if p.sessionIDs != 1 || p.agentIDs > 1 {"
  evidence: internal/core/history/ingest_test.go:364 — "TestIngestSkipsAFileThatIsNotOneTranscript"
- cond-2609091722231767 — survived: Ownership comes only from the cwd values the transcript's own lines record, resolved per session first and per file only as a fallback, and the harness project name is kept as an opaque label that is never decoded to a path.
  evidence: internal/core/history/ingest.go:226 — "placeSessions resolves the owning repository once per SESSION"
  evidence: internal/core/history/ingest.go:311 — "Fallback: the file's own recorded directory."
  evidence: internal/core/history/ingest.go:396 — "the name AS GIVEN, never decoded into a path"
- cond-2609091722232975 — survived: A cwd is mapped to its repository's root-commit SHA, so a worktree keys to the store of the repository it derives from, and a session whose worktree is gone is still placed by its main thread or by the store.
  evidence: internal/core/history/ingest.go:79 — "A session run in a worktree resolves to the repository the worktree derives from, because they share a root commit."
  evidence: internal/core/history/ingest_test.go:107 — "Only the parent's directory resolves; the worktree is gone."
  evidence: internal/core/history/ingest_test.go:143 — "TestIngestPlacesASessionFromTheStoreWhenNoDirectorySurvives"
- cond-2609091722237552 — survived: The scanner is constructed from the destination root alone and Capture fails closed on a surviving blocking span, and ingest files that refusal into Failed where the front door prints it rather than working around it.
  evidence: internal/core/history/history.go:212 — "sc, err := scanner.New(repoRoot)"
  evidence: internal/core/history/ingest.go:377 — "res.Failed = append(res.Failed, IngestFailure{Path: p.path, Err: err.Error()})"
  evidence: internal/surface/cli/history_recovery.go:302 — "FAILED %s: %s"
- cond-2609091722234418 — survived: Core refuses to run with no source and knows no vendor path; the only sources are the operand paths or the destination's declared ingest_roots, and the surface's single piece of harness knowledge is a file NAME, never a directory layout.
  evidence: internal/core/history/ingest.go:172 — "ingest needs at least one source path; declare them in"
  evidence: internal/core/history/config.go:46 — "This is the ONLY place a transcript-source path lives"
  evidence: internal/surface/cli/history_recovery.go:33 — "The NAME is used, never the directory structure around it"
- cond-2609091722233051 — narrowed: The one-transcript-resident invariant holds and the backlog scale was actually run, but the delivered code reads a transcript the destination owns TWICE - once whole in the probe and again in store before Capture - so the each-file-read-once half of the assumption does not hold as written.
  narrowing: Holds for the memory invariant (one transcript resident at a time) and at the ~1100-transcript scale the recovery run reached, but not for 'each file read once': every file is read whole by probeTranscript, and every file actually stored is read a second time by store() before Capture, so an owned transcript costs two full reads rather than the bounded prefix plus one.
  evidence: internal/core/history/ingest.go:499 — "raw, err := fsutil.ReadGuarded(c.path, maxTranscriptBytes)"
  evidence: internal/core/history/ingest.go:369 — "Re-read: the probe kept the transcript's identity, not its bytes."
  evidence: internal/core/history/ingest.go:498 — "The whole file is read, and the bytes are DISCARDED"
## Grounds

- pursued: we expect a defaultless destination operand plus per-session-before-per-file owner resolution from the cwd recorded inside a transcript to make backlog recovery safe and worth running, because the scanner is built from the named destination's repository root and from nothing else, and because a session whose worktrees the harness has deleted is still placed by its main thread or by the store that has seen it; it is shown wrong if transcripts' recorded working directories do not resolve often enough for the recovery to be worth running, if sessions routinely record two repositories so that the ambiguity refusal swallows the corpus, or if a transcript is ever stored under a repository's redaction configuration that is not the one the operator named
