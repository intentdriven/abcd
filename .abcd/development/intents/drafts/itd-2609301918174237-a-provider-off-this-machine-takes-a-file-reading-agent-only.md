---
id: itd-2609301918174237
slug: a-provider-off-this-machine-takes-a-file-reading-agent-only
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609081951381895, itd-2609170822093401]
severity: minor
origin: researcher-authored
production_mode: hand-written
related_adrs: [adr-2609221009491186, adr-2609300107513982]
---

# A remote provider takes a file-reading agent only as a fixed, blanked, capped bundle

## Press Release

> **A provider off this machine now takes a file-reading agent's work as one bundle: a fixed list of files for that kind of helper, scanned and blanked before it leaves, and held under a size cap the owner sets per project on their own machine.** The intent auditor, the consistency pass and the release composer could never run on a paid provider, because a provider call carries no tools and nothing built what they read. Now each kind of helper has a fixed list (the auditor gets the request, the intent, its specs, the delivered diff and the changed files), the owner can change that list and the cap for one project in `~/.abcd.noindex/config.json` and nowhere else, every secret-shaped span is blanked to a placeholder with the call record saying what was blanked, and a bundle over the cap is refused rather than cut short. The same rules hold for every provider off this computer, keyed or not; a server on this machine is outside them.
>
> "I pointed the auditor at a cheap reviewer model for one project and left my other projects alone," said a product thinker who pays for an aggregator. "I could see exactly which files went, the record told me two lines were blanked, and when a big delivery went over my cap it stopped and told me the number instead of sending half of it."

## Why This Matters

The OpenAI-compatible adapter (itd-2609081951381895) admits only self-contained agents to a provider that holds a key (ruling DR5 of 2026-09-29), and its Decision 11 left every file-reading agent refused because which files travel, a size cap and a scan before sending were choices no ruling made. The product thinker ruled all three on 2026-09-30 (DR5b-1 to DR5b-3), and ruled that the rules follow the provider's location rather than its key (DR5b-4), which closes the keyless remote route the adapter's security review found: a keyless https provider on another host is outside DR5 as built, so a checkout's own route can aim the auditor's request, the consistency corpus and the release cut at it. The same review found the cold reading's `unscanned` bundle items travel whole under the person's key; the send-time scan covers them too.

## Mechanism

We expect a fixed list per helper kind, blanked and capped, to let the file-reading helpers run on a provider without sending anything the owner did not choose, because the list, the cap and the override all live in the owner's machine settings where no checkout or agent writes, and the blanking runs through the one scanner every other outbound path already trusts; shown wrong if a dispatched bundle ever carries a file outside its kind's list or the owner's override, a secret the canonical scanner detects, or more bytes than the cap in force.

## Scope Conditions

- Holds for the helpers a verb already dispatches to a provider (the intent auditor in both roles, the release composer, the four cold-reading positions); an agent no verb dispatches gains no list here.
- Holds where the project has a root commit: a repository with no commit has no per-project key, and only the machine's defaults apply to it.
- Holds for secrets the canonical scanner's pattern set detects; a secret shape outside that set is not blanked, as on every other outbound path.

## What's In Scope

- **One classification, "remote".** A provider is remote when its `base_url` host is not this machine (not `localhost` and not a loopback address, judged from the configured URL's text with no name lookup), whether or not its block names a key. DR5, its override and every rule below apply to a remote provider; a provider on this machine is outside them, keyed or not.
- **A fixed input list per helper kind**, compiled into the binary, grounded in what each helper reads today: `intent_audit`, `intent_consistency`, `release_changelog`, and the cold reading's `reading` kind.
- **The owner's per-project override** of a kind's list and of the cap, in `~/.abcd.noindex/config.json` under `oracle.bundles`, keyed by the project's root-commit SHA; a default cap there for every project.
- **A repository's `oracle.bundles` ignored with a warning**, the rest of its configuration loading.
- **A bundled default cap, and a refusal over the cap**, never a truncation.
- **Scan and blank** of every bundle item through the canonical scanner before the send, the blanked spans named in the call record.
- **The reading bundle's `unscanned` items scanned** at the send, and the count said on stderr before it.
- The bundle's kind, items, size, cap and blanked spans on the call record the receipt already carries.

## What's Out of Scope

- A provider on this machine: it keeps today's dispatch (the emitted request files; the parked reading bundle), unscanned and uncapped.
- The contents of a cold reading's bundle: `reading assemble` and its include preset decide them (itd-2609081951381895 Decision 10 admits the positions as self-contained); this intent scans and caps that bundle at the send and changes nothing it holds.
- The four disembark agents, which no verb dispatches to a provider (itd-2609081951381895 Decision 13), and the agents no verb dispatches at all.
- Chunking or summarising an over-cap bundle.
- The machine layer's trust in `$HOME` (iss-2609300012273350), which the override inherits.

## Decisions

Ruled by the product thinker on 2026-09-30, relayed by abcd-a2 [75597f] and recorded by lane dr5bSpec of autonomous run A, answering the STOP the wiring lane of itd-2609081951381895 raised on bundles (DR5, 2026-09-29, verbatim in the decision log: "only self-contained agents go to a paid provider by default, with a central machine-level override list the person keeps"):

1. **DR5b-1, which files travel:** "(b) A FIXED LIST PER HELPER KIND (e.g. the reviewer gets the changed files plus the spec)". Asked whether it is configurable per repository, the person ruled that the lists change ONLY in the owner's personal (machine-layer) settings, per project, and a repository's setting is ignored with a warning.
2. **DR5b-2, the size cap:** "that's only for abcd, not for other projects; each project owner will have their own external services." The choice: THE OWNER'S PERSONAL SETTINGS, PER PROJECT, meaning a default cap plus per-project caps in the machine layer; repository settings are ignored for this.
3. **DR5b-3, a scan before sending:** "(a) CHECK AND BLANK OUT: scan, redact to a placeholder, send, and note what was blanked."
4. **DR5b-4, which providers the rules govern:** "TREAT ANY REMOTE PROVIDER LIKE A PAID ONE: every provider off this computer (keyed or keyless) needs the owner's personal list and gets the fixed file list, the owner's cap and blanking; only on-machine helpers skip these rules."

Taken by the technical facilitator in lane dr5bSpec (2026-09-30), within the rulings above:

5. **The home is a new intent, not spc-2609251028149555.** itd-2609170822093401 puts the adapters and what they send out of its scope, and names itd-2609081951381895 as their home; that intent has shipped, and its remainder spec is blocked on the implement loop's state file for its AC 10, which would hold this work back for a reason unrelated to it.
6. **"Per project" is keyed on the root-commit SHA**, the full object name `gitutil.RootCommit` returns, as the transcript and worktree stores are keyed: a checkout moves, is renamed, is cloned twice and has several worktrees, and the root commit changes under none of that. The cost is that every clone and fork of one history shares its entry.
7. **Over the cap is a refusal, not a truncation.** DR5b-1 lets only the owner change a kind's list, and cutting a bundle short would be abcd changing it; DR5b-2 makes the cap the owner's statement of what their own external service takes, so a bundle over it is the owner's to decide on (raise the project's cap, trim the project's list, or keep the run on the host), and a helper judging a truncated delivery would return a verdict on evidence it never saw.
8. **The bundled default cap is 4 MiB** (4194304 bytes, `issueschema.RunArtefactReadLimit`, the largest run artefact abcd already reads, so a parked reading bundle that dispatches today is not refused by the default); the owner lowers it for every project or for one.
9. **The keyed-provider rules of itd-2609081951381895 Decision 8 cover a remote provider too.** A repository's route to a remote provider is skipped with the CD2 diagnostic, and a repository row's settings on a remote leg are refused, as on a keyed one: DR5b-4 treats any remote provider like a paid one. A repository's route to a provider on this machine that names no key is still admitted.

## SOTA

Declared by lane dr5bSpec on 2026-09-30 from the record and the tools abcd
already carries. No independent research pass has checked it, so the
maturities below are rough and are owed a primary-source check before the
build adopts anything outside the tree:

- **Secret detection before text leaves.** Mature, dedicated scanners exist,
  gitleaks the most widely used, and abcd already runs it as a repository's
  opt-in augmenter inside its own scanner (`scanner.WithAugmenter`). Path 2:
  the canonical scanner is the native floor and the augmenter seam is the
  way a stronger detector plugs in, so the blanking adds no dependency.
- **Choosing what context a model is sent.** Coding assistants select context
  by retrieval or a repository map, and hosted assistants let an organisation
  exclude paths from what is sent. Neither is a fixed, owner-keyed list per
  task that a repository cannot widen. Path 2: a compiled list per kind with
  the owner's per-project override is the floor, and a selector could later
  fill a kind's list behind the same input vocabulary.
- **Request size limits.** Providers enforce context windows and bill by
  token. A byte cap judged before the call is the floor. A token estimate
  (the reading's `tokens_est`) could replace it behind the same refusal.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** a provider block whose `base_url` names a host other than this machine and no key, and a machine route pointing the intent auditor at it, **when** `abcd intent audit` runs with the provider absent from `oracle.bundled_context_providers`, **then** it is refused before any call, naming DR5 and the override; the same block at a loopback address takes the step as today.
- **Given** a repository `.abcd/config.json` routing the intent auditor to a remote provider that names no key, **when** the configuration is read, **then** the route is skipped with a diagnostic naming the route and `~/.abcd.noindex/config.json`, and the machine's own route applies.
- **Given** a remote provider named in `oracle.bundled_context_providers` and the auditor routed to it, **when** a fidelity review with a delivered range is sent, **then** the provider receives exactly the `intent_audit` list in its order (the request, the intent, each spec the request names, the range's diff, each changed file at the range's head), and the call record names the kind and every item.
- **Given** a fidelity review whose caller holds no delivered range, on a branch with no commit beyond the default branch, **when** it would be sent to a remote provider, **then** it is refused before any call, naming the missing range and `--route intent-auditor=host-decides`.
- **Given** `oracle.bundles.projects.<root-sha>.kinds.intent_audit` in `~/.abcd.noindex/config.json`, **when** a review is sent from that project, **then** the bundle carries that list; from another project, it carries the default list; an entry outside the kind's inputs that is not a `path:` pattern refuses the configuration read, naming the entry.
- **Given** a repository `.abcd/config.json` declaring `oracle.bundles`, **when** a delegating verb reads the configuration, **then** one stderr line names the file, the key and `~/.abcd.noindex/config.json` as where it is set, the declaration changes nothing, and the verb continues.
- **Given** a bundle larger than the cap in force (the project's, else the machine's default, else 4 MiB), **when** it would be sent, **then** it is refused before any call, naming its size, the cap, where the cap came from and the key that raises it for this project, and nothing is truncated or written.
- **Given** a bundle item carrying a secret the canonical scanner detects, **when** it is sent to a remote provider, **then** the provider receives a placeholder in its place, one stderr line before the send counts what was blanked, and the call record names each blanked span's item, line and kind, never its value.
- **Given** a scanner whose configuration is degraded, or a blocking finding that survives the blanking, **when** a bundle would be sent, **then** it is refused before any call.
- **Given** a parked cold reading whose manifest counts `unscanned` items, **when** `reading ingest --dispatch` sends it to a remote provider, **then** every item is scanned and blanked, stderr says how many items were unscanned at assembly and are scanned now, and the call record carries the counts.
- **Given** the release composer or the consistency pass routed to a remote provider on the owner's list, **when** the verb sends its step, **then** the provider receives exactly that kind's list, blanked and within the cap.
- **Given** an agent with no kind (a disembark agent, or any agent without a list), **when** it is routed to a remote provider on the owner's list, **then** it is refused before any call.
- **Given** the lane, **when** it ships, **then** a security review of the bundle, the blanking and the remote classification is on its record.

## Audit Notes

Filed 2026-10-02 as a draft pending two adversarial reviews and a full planning interview, under the product thinker's ruling DR5c-0 of 2026-10-02: lane dr5bSpec first wrote it as planned without that interview, and nothing is built from it meanwhile. The spec that lane drafted was not filed; a parked copy of it is kept in the local tier for the interview. The kind, the impact and the scope-condition identities are left for the planning interview to set.

## Grounds

- pursued: the owner decides per project what a remote helper may read and how much, on their own machine, and every byte that leaves is scanned by the one scanner; we expect the file-reading helpers to run on a provider with no file outside their list, no detected secret and no bundle over the cap leaving the machine; shown wrong if a call record or a provider fake ever shows otherwise.
