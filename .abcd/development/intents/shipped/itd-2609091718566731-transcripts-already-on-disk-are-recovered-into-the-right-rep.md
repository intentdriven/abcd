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

<!-- abcd-review: OWED receipt=rcp-813aa9d78674 -->
Fidelity review OWED (receipt rcp-813aa9d78674).

## Grounds

- pursued: we expect a defaultless destination operand plus per-session-before-per-file owner resolution from the cwd recorded inside a transcript to make backlog recovery safe and worth running, because the scanner is built from the named destination's repository root and from nothing else, and because a session whose worktrees the harness has deleted is still placed by its main thread or by the store that has seen it; it is shown wrong if transcripts' recorded working directories do not resolve often enough for the recovery to be worth running, if sessions routinely record two repositories so that the ambiguity refusal swallows the corpus, or if a transcript is ever stored under a repository's redaction configuration that is not the one the operator named
