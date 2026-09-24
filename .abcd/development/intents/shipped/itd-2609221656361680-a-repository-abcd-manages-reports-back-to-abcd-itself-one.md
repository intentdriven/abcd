---
id: itd-2609221656361680
slug: a-repository-abcd-manages-reports-back-to-abcd-itself-one
spec_id: spc-2609221657168936
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-4]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-2609151838312703, itd-33, itd-116]
related_adrs: [adr-2609221009491186]
---

# A managed repository reports back to abcd, and abcd says so at its next start

## Press Release

> **A repository abcd manages files an enhancement proposal or a defect report against an abcd-issued template, into an inbox in the user account; abcd says at its next start how many wait and from how many repositories.**
>
> "Every useful thing my managed repositories learned about abcd reached me because an agent happened to mention it in a message, and I filed it by hand at midnight," said a product thinker running abcd on three repositories. "Now the repository files it, the report waits in my own account, and abcd greets me with one line saying three are waiting. I read them when I choose; nothing files itself."

## Why This Matters

Every managed repository already has a `for_abcd/` folder in its local tier, and nothing in abcd reads or writes it: on 2026-09-21 three accounts of a sibling repository's runs reached abcd only because an agent relayed them in a message and a person filed the findings by hand at the end of a long day. A report that lives in the managed repository's own tree reaches abcd only if someone carries it; a report in the user account's machine store, where abcd already keeps its history, transcripts, worktrees, sources and labs, is where abcd can find it by itself.

## Mechanism

We expect a reported finding to reach abcd's ledger where a relayed one is lost, because the report outlives the session that noticed it and abcd reads its own inbox without being asked; shown wrong if the inbox fills with reports nobody drains, or if findings keep arriving as chat messages after it ships.

## Scope Conditions

- Holds where the reporting repository and abcd are managed by the same user account on one machine; another account or another machine is the mailbox and broker drafts' question, not this record's. <!-- cond: cond-2609221657163508 -->
- Holds for a report small enough to write at the moment of the finding; a run's whole account is a document the report points at, not the report's body. <!-- cond: cond-2609221657162395 -->

## What's In Scope

- **The template**: abcd issues it (`abcd report --template` writes the skeleton), a machine-readable block (schema version, kind, severity, category, the abcd version and surface in play, a remedy where the reporter has one, evidence pointers) beside prose the reporter writes.
- **The verb**: `abcd report [<file>]` in the managed repository validates the template and files it; with no file it opens the skeleton for the reporter to fill.
- **Where**: `~/.abcd/inbox/<received-stamp>-<sender-key>.md` by default, the machine store keyed as the other stores are; never the managed repository's tree and never abcd's.
- **The greeting**: abcd's session-start line says how many reports wait and from how many repositories, once, and nothing else; the board's own row says the same.
- **The reading**: `abcd inbox` renders them newest first, naming the sender repository plainly, read-only; `abcd inbox show <id>` renders one.
- **The filing**: nothing is filed until a person or a session acts; `abcd inbox promote <id>` files it as a capture carrying the sender's FINGERPRINT (the repository's root-commit key) and a generic description, never its name, scanned before it is written, with the report's id as its evidence pointer.
- **The unknown version**: a report whose template version abcd does not know is listed as unreadable with the version named, and is never dropped or half-read.

## What's Out of Scope

- Cross-account and cross-machine delivery (the mailbox draft itd-2609151838312703 and the broker draft itd-2609151838327688).
- Any automatic filing.
- A reply channel: abcd's answer to a report is the record it files and the release that carries it.

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609221009491186 records the vocabulary rulings it rests on):

1. A written account against an abcd-issued template, with a machine-readable block beside the prose (ruled 2026-09-22).
2. It lands in the user account's machine store, not in either repository's tree.
3. abcd says it is there at its next start; nothing is filed until a person or a session acts.
4. Identity in the inbox, fingerprint in the record: the inbox names the sender plainly so the reader knows who is asking; anything committed carries the root-commit key and a generic description.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** a managed repository with a finding, **when** `abcd report` runs, **then** the report is validated against the abcd-issued template and filed, with its machine-readable block and its prose, and the verb names where it landed.
- **Given** any report, **when** it is filed, **then** it lands in the machine store's inbox under the user account and nothing is written into either repository's tree.
- **Given** reports waiting, **when** abcd starts a session, **then** one line says how many wait and from how many repositories, and the status board carries the same row.
- **Given** waiting reports, **when** `abcd inbox` runs, **then** they render newest first naming the sender repository, read-only, with nothing filed.
- **Given** a report, **when** nobody acts on it, **then** nothing is filed; **when** `abcd inbox promote` runs on it, **then** a capture exists carrying the sender's root-commit key, a generic description, the report's id as evidence, and no repository name, scanned before the write.
- **Given** a report whose template version abcd does not know, **when** the inbox renders, **then** it is listed as unreadable naming the version, and it is not dropped.
- **Given** the sender's name, **when** anything abcd commits is searched for it, **then** it is absent.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-ca83a0855a0d -->
Fidelity review — receipt rcp-ca83a0855a0d (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:e69022f8553e07416af18d81519a95c35c2242691ae8b2ce393a681fad16265b
Input attestations: diff:60d65063^1..60d65063 (PR #687, feat/managed-repo-reports-back)@sha256:b56f8b2abc0f3a34037db23af61a9b0b26fc93e2e1cb1b35f96acd9cdbc12c05;

Acceptance rollup: MET 4 · MET_WITH_CONCERNS 3 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: abcd report --template issues the skeleton, Parse validates a filled one against it (block and prose), File writes it, and the verb prints the id and the ~/.abcd/inbox path; the CLI test drives all three through stdin
  evidence: internal/core/report/report.go:158 — "func Template(abcdVersion string) []byte"
  evidence: internal/core/report/report.go:209 — "func Parse(data []byte) (Report, error)"
  evidence: internal/surface/cli/report.go:120 — "fmt.Fprintf(w, "filed %s — %s\n", filed.ID, termsafe.Sanitize(filed.Path))"
  evidence: internal/surface/cli/report_surface_test.go:83 — "!strings.Contains(out, "filed rpt-") || !strings.Contains(out, "~/.abcd/inbox/")"
- ac-2 — MET: File creates only under os.UserHomeDir()/.abcd/inbox by exclusive create; the core test asserts the sending repository's porcelain is empty after filing and the reported path carries no home directory; the editor path uses a system temp file, never a tree
  evidence: internal/core/report/inbox.go:23 — "const inboxRelPath = ".abcd/inbox""
  evidence: internal/core/report/inbox.go:219 — "err = fsutil.CreateExclusiveIn(root, name, serialize(r), fileMode)"
  evidence: internal/core/report/inbox_test.go:101 — "if st := repo.Git("status", "--porcelain", "--untracked-files=all"); st != """
  evidence: internal/surface/cli/report.go:142 — "f, err := os.CreateTemp("", "abcd-report-*.md")"
- ac-3 — MET: hook session-start prints inboxGreeting() as one stdout line built from Count() (reports, distinct sender keys); the board carries Inbox: boardInbox() as an inbox: row and a JSON object; the surface test asserts exactly one report(s) line, the wording, no sender name, and the same tally on the board
  evidence: internal/surface/cli/cli.go:1508 — "if g := inboxGreeting(); g != "" {"
  evidence: internal/surface/cli/report.go:209 — "return "abcd: " + inboxTallyText(t) + " wait in the inbox; `abcd inbox` lists them.""
  evidence: internal/surface/cli/cli.go:260 — "fmt.Fprintf(w, " inbox: %s — `abcd inbox`\n", inboxTallyText(*board.Inbox))"
  evidence: internal/surface/cli/report_surface_test.go:259 — "want := "abcd: 3 report(s) from 2 managed repositories wait in the inbox; `abcd inbox` lists them.""
- ac-4 — MET_WITH_CONCERNS: List reads through peekInbox (never creates), sorts newestFirst, and the CLI prints each readable entry's sender_name; tests assert order, the name, and an empty porcelain after reading. Concern: a waiting report the inbox cannot read is listed with a 12-hex key prefix and no sender name, so for that entry the list does not name the sender repository
  evidence: internal/core/report/inbox.go:358 — "func List() ([]Entry, error)"
  evidence: internal/core/report/inbox.go:343 — "func newestFirst(es []Entry)"
  evidence: internal/surface/cli/report.go:291 — "fmt.Fprintf(w, " %s %s %s %s/%s %s\n", e.ID, e.ReceivedAt, termsafe.Sanitize(e.SenderName),"
  evidence: internal/surface/cli/report.go:288 — "fmt.Fprintf(w, " %s %s key %s UNREADABLE: %s\n", e.ID, e.ReceivedAt, e.SenderKey[:12],"
  evidence: internal/surface/cli/report_surface_test.go:170 — "if strings.Index(list, second) > strings.Index(list, first)"
- ac-5 — MET_WITH_CONCERNS: Nothing is filed until Promote runs (the ledger's porcelain is empty after File); Promote composes a capture carrying SenderKey, GenericSender and the rpt id as evidence, and files it through capture.Capture, whose write path redacts text, slug, found-at and found-during first. Concerns: (1) the name scrub deliberately leaves a sender whose directory name is a common word or three letters or fewer in the prose as a word (brief 30-inbox.md:81), so 'no repository name' holds for distinctive names only; (2) promotion is accepted only in a checkout whose root commit is abcd's, a tightening beyond the press release's 'files it as a capture'
  evidence: internal/core/report/inbox_test.go:220 — "t.Fatalf("a report filed itself before anyone acted:\n%s", st)"
  evidence: internal/core/report/inbox.go:623 — ""\nReported by %s (root commit %s) through the abcd inbox as %s, a %s against abcd %s, surface %s.\n""
  evidence: internal/core/report/inbox.go:625 — "fmt.Fprintf(&b, "\nEvidence:\n\n- %s (the report, kept in the inbox)\n", id)"
  evidence: internal/core/capture/workflow.go:78 — "redactCaptureInputs(repoRoot, req.Text, req.Slug, req.FoundAt, req.FoundDuring)"
  evidence: internal/core/report/inbox.go:741 — "func commonName(parts []string) bool"
  evidence: internal/core/report/inbox.go:521 — "if root := gitutil.RootCommit(ledgerRoot); root != abcdRootCommit {"
- ac-6 — MET: parse judges schema_version before any other key, returns a VersionError naming the version, readEntry maps it to StateUnreadable with that message, and the core test plants a version-7 file and asserts it is listed naming 7, still on disk, counted, and refused by Promote
  evidence: internal/core/report/report.go:305 — "if version != SchemaVersion {"
  evidence: internal/core/report/report.go:103 — "return fmt.Sprintf("template version %s is not one this abcd knows (it reads version %d)", e.Version, SchemaVersion)"
  evidence: internal/core/report/inbox.go:276 — "case errors.As(err, &ve):"
  evidence: internal/core/report/inbox_test.go:180 — "t.Errorf("the unreadable report was dropped: %v", err)"
- ac-7 — MET_WITH_CONCERNS: captureRequest runs nameScrubber over every free-text field and FoundDuring uses GenericSender; the core test files under the name Zanzibar-Quartz written into title and prose and asserts no file name or content the ledger gained contains it, and spelling-variant tests cover separators, case, a trailing digit and forge addresses. Concern: the guarantee is lexical and, by design, exempts a name that is a common word or three letters or fewer (cli, api, docs, abcd), which then survives into the committed capture as a word
  evidence: internal/core/report/inbox.go:608 — "named := nameScrubber(r.SenderName)"
  evidence: internal/core/report/inbox_test.go:255 — "if strings.Contains(strings.ToLower(string(data)), "zanzibar") {"
  evidence: internal/core/report/inbox_test.go:532 — "if got := nameScrubber(tc.name)(tc.in); got != tc.in {"
  evidence: .abcd/development/brief/04-surfaces/30-inbox.md:81 — "A name that is a common word — three letters or fewer (`cli`,"

Gap audit:
- honoured:
  - abcd issues the template with a machine-readable block (schema version, kind, severity, category, abcd version, surface, remedy, evidence) beside prose
    evidence: internal/core/report/report.go:158 — "func Template(abcdVersion string) []byte"
  - abcd report validates and files; bare abcd report opens the skeleton in the reporter's editor
    evidence: internal/surface/cli/report.go:133 — "func reportFromEditor(skeleton []byte) (data []byte, kept string, err error)"
  - reports land at ~/.abcd/inbox/< received-stamp>-< sender-key>.md, never in either tree
    evidence: internal/core/report/inbox.go:218 — "name := stamp + "-" + s.Key + ".md""
  - the session-start greeting is one line of counts and nothing else, and the board's row says the same
    evidence: internal/surface/cli/report.go:199 — "func inboxTallyText(t report.Tally) string"
  - abcd inbox lists newest first naming the sender, read-only; abcd inbox show renders one
    evidence: internal/surface/cli/report.go:298 — "Use: "show < id>","
  - nothing is filed automatically; promote files a capture carrying the root-commit key and a generic description, with the report id as evidence, through capture's own redactor
    evidence: internal/core/report/inbox.go:578 — "res, err := capture.Capture(req)"
  - an unknown template version is listed unreadable naming the version and never dropped
    evidence: internal/core/report/inbox_test.go:174 — "list[0].State != StateUnreadable || !strings.Contains(list[0].Unreadable, "7")"
  - the filed report reaches abcd's own ledger (the intent's mechanism): promotion is refused in any other checkout
    evidence: internal/core/report/inbox_test.go:403 — "func TestPromoteRefusesOutsideAbcdsOwnCheckout(t *testing.T)"
  - wired on both surfaces: CLI verbs registered, plugin pages commands/report.md and commands/inbox.md, brief chapters 29 and 30
    evidence: internal/surface/cli/cli.go:280 — "root.AddCommand(newReportCommand(&asJSON))"
    evidence: .abcd/development/brief/04-surfaces/30-inbox.md:1 — "# `/abcd:inbox` — Read and Promote Reports"
- diverged:
  - the committed capture carries no repository name — narrowed: a sender named for a common word or three letters or fewer (cli, api, docs, abcd, unnamed) is not scrubbed as a word and survives into the capture; only its forge address is replaced
    evidence: internal/core/report/inbox.go:729 — "var commonNames = map[string]bool{"
    evidence: .abcd/development/brief/04-surfaces/30-inbox.md:81 — "A name that is a common word — three letters or fewer (`cli`,"
  - the inbox names the sender repository plainly so the reader knows who is asking — narrowed: an unreadable waiting report (unknown version or malformed) is listed with a 12-hex key prefix and no name, because the name sits in the envelope the failed parse never reaches
    evidence: internal/surface/cli/report.go:288 — "e.SenderKey[:12], termsafe.Sanitize(e.Unreadable)"
    evidence: internal/core/report/report.go:293 — "vl, ok := block[keyVersion]"
- missing: (none)

Scope-condition dispositions:
- cond-2609221657163508 — survived: the inbox is resolved under the caller's own home and a promotion needs abcd's checkout on the same machine; nothing crosses an account or a machine, which is what the condition assumed
  evidence: internal/core/report/inbox.go:144 — "func inboxDir() (home, dir string, err error)"
  evidence: internal/core/report/inbox.go:521 — "if root := gitutil.RootCommit(ledgerRoot); root != abcdRootCommit {"
- cond-2609221657162395 — survived: a report is bounded at 32 KiB with per-field bounds and up to twenty evidence pointers, and the refusal text sends a longer account to a pointer, so the report stays the small thing the condition assumed
  evidence: internal/core/report/report.go:54 — "const MaxBytes = 32 << 10"
  evidence: internal/core/report/report.go:232 — "(point at a longer account instead of pasting it)"
## Grounds

- pursued: three managed-repository accounts this week reached abcd only because an agent relayed them and a person filed them by hand; we expect a filed report to reach the ledger where a relayed one is lost; shown wrong if the inbox fills with reports nobody drains or findings keep arriving as chat messages
