---
id: spc-2610120557432880
slug: when-someone-reports-a-security-hole-in
intent: itd-2610112239022766
origin: researcher-authored
production_mode: hand-written
---
# when-someone-reports-a-security-hole-in

## Summary

This spec delivers the first rung of `abcd embargo` for
itd-2610112239022766: five sub-verbs that take a private GitHub security
advisory from intake to publication or closing without its record ever
entering the public ledger. Its trust rules are
[adr-2610120558328615](../../decisions/adrs/2610120558328615-a-private-advisory-s-record-never-enters.md)
(proposed) and the brief's invariant 22.

- `abcd embargo intake` lists the repository's advisories in the `triage` and
  `draft` states through GitHub's REST API, called with the person's own `gh`,
  and mints one private issue record per new advisory in the embargo store, a
  detached worktree in the machine's worktree store that is never committed.
- `abcd embargo triage <iss-N|GHSA-id> <verdict>` records one of four verdicts
  (`vulnerability`, `not-abcds`, `not-a-vulnerability`, `public-track`) with
  who decided it and when, and for the two closing verdicts drafts the
  explanatory text for the person to approve.
- `abcd embargo close <iss-N|GHSA-id>` shows the drafted closing text and
  changes nothing; with `--approve <digest>` of exactly that text it closes the
  advisory on GitHub with it and reads the advisory back (the product
  thinker's ruling of 2026-10-12 moved closing into this rung).
- `abcd embargo check` is the outbound embargo check: it reads a commit range,
  any number of text files (a pull-request body, a changelog section, a
  release payload, a record) and refuses every advisory id, reserved CVE id,
  advisory title and store path that would disclose an unpublished advisory.
- `abcd embargo publish` computes each fixed advisory's patched version from
  the first release tag containing its fix, shows the plan and changes
  nothing; with `--approve <plan-digest>` it publishes one advisory as a
  canary, reads it back, publishes the rest, and confirms none is left in
  draft.

The hand-runs of 2026-10-09 and 2026-10-11 were the script stage of
script-first-mvp: they found the contract (a private store worktree, a
GHSA-to-record map, neutral public text, a computed patched version, a canary
publication). This spec absorbs that contract into `internal/core/embargo`.
Lane running, batch landing with redaction, CVE requests, restoring redacted
records and the broken state are out of scope (see Out of scope).

| Criterion (intent order) | Met by |
| --- | --- |
| 1, intake mints private records, severity mapped, nothing public written | Steps 3 and 4 |
| 2, re-run does not duplicate; an advisory with no severity is listed, not recorded | Step 4 |
| 3, four verdicts with who and when; unknown or reason-less verdict refused | Step 5 |
| 4, closing verdicts draft the text; "not abcd's" names both halves of the rule | Step 5 |
| 5, without approval nothing is posted; with approval of exactly the shown text the advisory is closed with it and re-read | Step 6 |
| 6, check refuses ids, reserved CVE ids, titles and the store path; clean text passes | Step 7 |
| 7, publish shows the plan with computed versions and changes nothing | Step 8 |
| 8, an unreleased or unresolved fix is left out of the plan with the reason | Step 8 |
| 9, approval-bound canary publication, read back, rest, no drafts left | Step 9 |
| 10, the command page says what stays the person's act | Step 11 |

## Footprint

- packages: internal/adapter/github (new; or the name the facilitator picks), internal/core/ahoy, internal/core/issueschema, internal/core/embargo (new), internal/core/peers, internal/surface/cli, internal/core/surface, commands/embargo.md, .abcd/rules.json
- tests: per step below; every remote call is exercised against a fake `gh` on PATH serving fixture JSON with reserved example ids, and every safety property is held by a named test.

## Design

### The embargo store

- Path: `abcdhome.Path(home, "worktrees", <root-sha>, "embargo")`, that is
  `~/.abcd.noindex/worktrees/<root-sha>/embargo/`, per
  adr-2610031751065746. `<root-sha>` is `gitutil.RootCommit` of the checkout,
  so every worktree of the repository finds the same store.
- Created on the first `intake` with `git worktree add --detach <store>
  <origin default branch tip>`, through the isolated git env. Never on a
  branch, so nothing in it has a ref to push.
- Private records are written by `capture.Capture` with `RepoRoot` set to the
  store, so they land in `<store>/.abcd/work/issues/open/` exactly where a
  lane or `capture resolve` reads them (as the hand-runs needed). They stay
  untracked.
- Store state: `<store>/.abcd/.work.local/embargo/state.json` (gitignored in
  every worktree of the repository), mode 0600, written atomically
  (temp file and rename), read with `jsonstrict` so an unknown key refuses:

  ```json
  {
    "schema_version": 2,
    "repo": "<owner>/<name>",
    "intake_at": "2026-10-11T22:00:00Z",
    "advisories": [
      {
        "ghsa_id": "GHSA-xxxx-xxxx-xxxx",
        "issue_id": "iss-2610112200001234",
        "title": "<the advisory summary>",
        "cve_id": null,
        "severity": "major",
        "forge_state": "draft",
        "verdict": null,
        "decided_by": null,
        "decided_at": null,
        "reason": null,
        "closed_at": null,
        "published_at": null
      }
    ],
    "pending": [
      {
        "ghsa_id": "GHSA-wwww-wwww-wwww",
        "title": "<the advisory summary>",
        "cve_id": null,
        "forge_state": "triage"
      }
    ]
  }
  ```

  The link from advisory to record lives here, not in the record: the record
  carries no advisory id, so the record a fix lands with has one less thing to
  redact.
- `VerifyStore` runs before every sub-verb that reads or writes the store and
  refuses (exit 2, nothing written) when: the store's HEAD is not detached;
  `git -C <store> ls-files` lists any record path the state names; the store
  path resolves inside the current checkout; `state.json` is not a regular
  file this account owns with no group or other write bit; or the store's
  folder, or a record the state names, carries any group or other permission
  bit. The store's folder is set to 0700 when it is made and each private
  record to 0600 when it is minted or moved, since git makes the worktree at
  the umask and capture writes a record at 0644.

### How `gh` is called

Every call goes through one runner, lifted out of `internal/core/ahoy/remote.go`
in step 1 (one-canonical-primitive): `gh` on PATH or a refusal carrying the
tool registry's install explanation; `--hostname github.com` on every call;
30-second timeout; stdout and stderr each capped at 1 MiB; request bodies on
stdin as JSON (`--input -`), never as `-f` bracket fields. The repository is
parsed from the `origin` URL by the same github.com-only parser. abcd holds
no token: the write is the person's own (adr-44).

| Purpose | Request | Scope needed |
| --- | --- | --- |
| Intake: list | `gh api --hostname github.com --paginate -X GET repos/{owner}/{repo}/security-advisories -f state=triage -f per_page=100`, then the same with `state=draft` | Classic token: `repo` or `repository_advisories:read`. Fine-grained: Repository security advisories, read. The account must be an admin or security manager of the repository, or a collaborator on the advisory. |
| Publish: read one | `gh api --hostname github.com repos/{owner}/{repo}/security-advisories/{ghsa_id}` | as above, read |
| Publish: set versions | `gh api --hostname github.com -X PATCH repos/{owner}/{repo}/security-advisories/{ghsa_id} --input -` with `{"vulnerabilities":[...]}` (and `"state":"draft"` when the advisory is still in `triage`) | Classic: `repo` or `repository_advisories:write`. Fine-grained: Repository security advisories, write. |
| Publish: publish | the same PATCH with `{"state":"published"}` | as above, write |
| Publish: confirm | the list request with `state=draft`, then `state=triage` | read |
| Close: read | the single GET above | read |
| Close: post and close | the PATCH above with `{"state":"closed","description":...}` | write, as for publish |

A `gh` failure whose stderr names a missing scope or a 403/404 is reported
with the scope line from this table and `gh auth refresh -s
repository_advisories:write` as the remedy; abcd never runs `gh auth` itself.

### Severity mapping

`critical` -> `critical`, `high` -> `major`, `medium` -> `minor`, `low` ->
`minor`. A null or unknown severity is not mapped: the advisory is listed
under `needs_severity` and no record is minted (unrecognized-input-never-writes).
It is still under embargo, so the state holds it in a `pending` list (its id,
title, reserved CVE id and forge state, no record), rebuilt from each intake's
listing, and the outbound check's deny set includes it. The pending list is
schema 2; a schema-1 state is read as one with nothing pending.

### Record shape at intake

`capture.CaptureRequest{RepoRoot: store, Category: security, Source:
security-advisory, Severity: <mapped>, Slug: "private-report", FoundDuring:
"embargo intake <UTC date>", Remedy: issueschema.MachineRemedy, Text:
<summary, then description>, Match: nil}`. The slug is neutral on purpose: a
fixed record lands under its file name, and a name derived from the advisory
title would disclose it. The body is the advisory's own text, private, and
passes through the capture redactor like every capture.

`security-advisory` is a new value of `issueschema.Sources` (step 2): the
ledger's source enum is the surfacing channel, and this is a channel the enum
does not yet name.

### Verdicts

| Verdict | Required flags | Store effect | Draft |
| --- | --- | --- | --- |
| `vulnerability` | `--by` | record stays in `open/` | none |
| `not-abcds` | `--by`, `--exists-without "<why>"`, `--cannot-help "<why>"` | record to `wontfix/` in the store via `capture.Wontfix`, reason composed from both | closing text |
| `not-a-vulnerability` | `--by`, `--reason "<why>"` | record to `wontfix/` in the store | closing text |
| `public-track` | `--by`, `--reason "<why>"` | record to `wontfix/` in the store, reason "moved to the public track" | none; prints the `abcd capture` command for the person to run in the public checkout |

`--by` is free text naming the person who decided (the store is private).
`decided_at` is stamped from the clock, never typed. A second triage of
the same advisory appends to a `history` array on the entry rather than
overwriting. The closing text is written to
`<store>/.abcd/.work.local/embargo/close/<ghsa_id>.md` and printed; posting it
is `abcd embargo close` (below), on the person's approval.

### Closing an advisory

`abcd embargo close <iss-N|GHSA-id> [--approve <digest>]`, for an advisory
whose current verdict is `not-abcds` or `not-a-vulnerability` and whose
`closed_at` and `published_at` are null; any other refuses with nothing sent.

GitHub's REST API has no comment on a repository advisory: the update's body
takes `summary`, `description`, `cve_id`, `vulnerabilities`, `cwe_ids`,
`credits`, `severity`, `cvss_vector_string`, `state` (`published`, `closed`,
`draft`) and the collaborators, and no endpoint of the group creates a comment.
The approved explanation is therefore posted as the head of the advisory's
`description`, above the reporter's own text, which is kept whole under a
separator line; the reporter's text is never dropped.

Without `--approve`: read the advisory (GET), read the drafted text from the
store, and show exactly the text that will be posted, the advisory's current
state, and `approve with: abcd embargo close <id> --approve <digest>`, where
the digest is SHA-256 over the drafted text's bytes and the digest of the
description as read. Nothing but the GET is sent.

With `--approve <digest>`: re-read the draft and the advisory and recompute;
a digest that does not match (the draft was edited, the advisory changed on
GitHub, or the approval was for another text) refuses with nothing sent and
says to re-run without `--approve` to see the current text. On a match: one
PATCH with `{"state":"closed","description":"<text>\n\n---\n\n<original>"}`,
then a GET that requires `state=closed` and a description that begins with
exactly the approved text; a mismatch is reported as a failed close, naming
which half did not hold. On success `closed_at` is stamped in the state.

The text is scanned before it is shown, with the same outbound scanner and
private-names banlist the publication plan uses: the advisory stays private
after closing, but its reporter reads the text, so a secret or a private name
in it refuses the show and the close alike. As with publication, the command
page instructs the host to pass `--approve` only after the person, in that
session, has seen the text and said yes to it.

### The outbound check

Inputs: `--range <a>..<b>` (commit messages from `git log --format=%B`, and
added lines from `git log -p --no-ext-diff --no-textconv`, through the
isolated git env so no repository-configured driver runs), and `--file
<path>` (repeatable; `-` reads stdin, capped at 1 MiB).

The deny set, built from `state.json`, for every advisory whose
`published_at` is null (a closed advisory stays private on GitHub, so it stays
in the set):

1. its `ghsa_id`, matched case-insensitively;
2. its `cve_id`, when set;
3. its `title`, whitespace-normalised and matched case-insensitively, until
   `closed_at` is set: the reporter chose it and nothing will publish it, so a
   closed advisory's title held forever (one such as "Merge pull request")
   would refuse every range after it, while its id and CVE id stay;
4. the store path in three forms: absolute, `~/`-relative, and the
   `worktrees/<root-sha>/embargo` fragment; and `.abcd/.work.local/embargo`.

Any other `GHSA-xxxx-xxxx-xxxx`-shaped id is refused too unless `--allow
<GHSA-id>` names it (an advisory of another project, already public), and
every allowed id is listed in the output. Findings carry kind, source
(`commit <sha>`, `file <path>`) and line, with the matched span masked the
way `scanner.Finding` masks it. Exit 0 clean, exit 2 on any finding. No store
present: exit 0 with one line saying nothing is under embargo on this machine.
The check also prints how long ago intake last ran, and warns when it is more
than 24 hours.

Known limit, stated on the command page: the check cannot recognise a
paraphrase or a count ("fixes several private reports"); that aggregate case stays the
author's to avoid.

### Publication

`abcd embargo publish [--release <vX.Y.Z>] [--approve <digest>]`. The release
defaults to the highest `vMAJOR.MINOR.PATCH` tag.

Plan, for every advisory whose verdict is `vulnerability` and whose
`published_at` is null:

1. Find `iss-N-*.md` under `.abcd/work/issues/resolved/` in the release tag's
   tree (`git ls-tree`). Absent: left out, "not resolved in <release>".
2. Read its `resolved_by.commit` from the tag's copy (`git show
   <tag>:<path>`). Absent or unreachable: left out with the reason.
3. First release tag containing the commit: `git tag --contains <sha> --list
   'v*'`, filtered to strict `vMAJOR.MINOR.PATCH`, sorted by semver, lowest
   wins. None: left out, "fix is in no release tag". A fix first released in
   v0.9.0 publishes as patched in 0.9.0 even when v0.10.0 is the release being
   cut; a hand-run typed the later version at first.
4. Read the advisory (GET). Take `vulnerabilities[].package` as the advisory
   carries it; no package: left out, "set the package on GitHub". Patched
   version: the tag without its leading `v`, unless the version form pinned in
   step 7 says otherwise. Affected range: `< <patched>`, keeping an existing
   lower bound (`>= a, < <patched>`). Both strings are validated against
   GitHub's range grammar (`=`, `<`, `<=`, `>`, `>=`, comma-and-space
   separated) before they enter the plan.
5. Scan the advisory's summary and description with the outbound scanner, the
   private-names banlist and the other-home-path rule; a finding leaves the
   advisory out, naming the finding kind.

The plan renders each included advisory (id, record, fix commit, first tag,
patched version, range) and each left-out one with its reason, and ends with
`approve with: abcd embargo publish --release <tag> --approve <digest>`, where
the digest is SHA-256 over the canonical JSON of the included entries, the
release tag and each advisory's text digest. Without `--approve` nothing is
sent but GETs.

Execution, with `--approve <digest>`: rebuild the plan; a digest that does
not match refuses with nothing sent (the plan changed, or the approval was
for another one). Then, in id order: the first advisory is the canary (PATCH
versions, GET and compare, PATCH published, GET and require
`state=published` and the planned versions); any mismatch stops the run with
nothing further sent. Then each remaining advisory the same way, stopping at
the first failure and reporting what is published and what is not. Then list
`state=draft` and `state=triage` and require that no planned id appears.
Each success stamps `published_at` in the state. A re-run after a partial
run plans only what is still unpublished.

The command page instructs the host: `--approve` is passed only after the
person, in that session, has seen the plan and said yes to it; the agent
never supplies it on its own.

## Steps

1. One `gh` runner
   - packages: internal/adapter/github (new; or the package name the facilitator picks); internal/core/ahoy uses it
   - needs: none
   - moves runGH, limitedWriter, githubHost and parseGitHubRemote out of ahoy/remote.go with no behaviour change
   - tests: ahoy's remote tests pass unchanged; TestRunnerNamesHostExplicitly (fake gh on PATH records argv); TestRunnerCapsOutput; TestRunnerRefusesWithoutGh; TestParseRemoteRefusesNonGitHub
2. The `security-advisory` source
   - packages: internal/core/issueschema, the issue JSON schema, record-lint's mirror
   - needs: none
   - tests: TestCaptureAcceptsSecurityAdvisorySource; record-lint accepts a record with it; an unknown source still refused
3. The embargo store and its state
   - packages: new internal/core/embargo (store.go, state.go)
   - needs: none
   - the state carries `closed_at` beside `published_at`, so step 6 adds no schema change
   - tests: TestStoreCreatedDetachedOnce (second open reuses it); TestVerifyStoreRefusesBranchHead; TestVerifyStoreRefusesTrackedRecord; TestVerifyStoreRefusesStoreInsideCheckout; TestStateRoundTrip; TestStateRefusesUnknownKey; TestStateWrittenOwnerOnly
4. Intake
   - packages: internal/core/embargo (intake.go)
   - needs: 1, 2, 3
   - tests (fake gh serving fixture JSON with reserved example ids, per examples-use-reserved-identifiers): TestSeverityMapping (table: critical, high, medium, low, null); TestIntakeMintsPrivateRecordInStore; TestIntakeLeavesPublicLedgerUntouched (the checkout's tree and index identical before and after); TestIntakeIsIdempotent; TestIntakeListsNullSeverityAndMintsNothing; TestIntakePaginates (two pages)
5. Triage
   - packages: internal/core/embargo (triage.go, closing.go)
   - needs: 3
   - tests (state seeded directly, so the step runs without intake): TestTriageEachVerdict; TestTriageRefusesUnknownVerdictWritingNothing; TestTriageRefusesMissingBy; TestNotAbcdsNeedsBothHalves; TestClosingDraftNamesBothHalves; TestRetriageAppendsHistory; TestPublicTrackPrintsCaptureCommand
6. Closing an advisory on GitHub
   - packages: internal/core/embargo (close.go)
   - needs: 1, 3
   - posts the approved closing text with one REST PATCH of the advisory, `{"state":"closed","description":"<approved text>, separator, the reporter's original description"}`; refuses, sending nothing, without `--approve` or with a digest that is not the one over exactly the text shown (and the description as read); then re-reads the advisory and requires `state=closed` with the description beginning with exactly that text
   - tests (fake gh recording each call; state and draft seeded directly): TestCloseWithoutApproveSendsOnlyGet; TestCloseWrongDigestSendsNothing; TestCloseEditedDraftInvalidatesApproval; TestCloseAdvisoryChangedOnGitHubInvalidatesApproval; TestClosePatchesStateClosedWithApprovedText; TestCloseKeepsReporterDescriptionBelowSeparator; TestCloseReadsBackAndReportsMismatch; TestCloseStampsClosedAt; TestCloseRefusesVulnerabilityOrUntriagedVerdict; TestCloseRefusesAlreadyClosedOrPublished; TestCloseScanFindingRefusesShowAndClose
7. The outbound check
   - packages: internal/core/embargo (check.go)
   - needs: 3
   - tests: TestCheckRefusesUnpublishedGHSA; TestCheckRefusesClosedUnpublishedGHSA; TestCheckPassesPublishedGHSA; TestCheckRefusesReservedCVE; TestCheckRefusesTitleCaseInsensitive; TestCheckRefusesStorePathForms; TestCheckRefusesUnknownGHSAUnlessAllowed; TestCheckReadsRangeMessagesAndAddedLines; TestCheckRunsNoRepoDiffDriver (a configured textconv must not run); TestCheckMasksSpan; TestCheckNoStorePasses
8. The publication plan
   - packages: internal/core/embargo (plan.go, versions.go)
   - needs: 1, 3
   - before writing versions.go, read one already-published advisory of this repository (GET with `state=published`) and pin the version form (with or without `v`) in one constant whose comment names that observation; TestPatchedVersionForm asserts it
   - tests: TestFirstReleaseTagContainingFix (repo fixture with v0.9.0 and v0.10.0, fix in v0.9.0, patched 0.9.0); TestPrereleaseTagsIgnored; TestFixInNoTagLeftOut; TestUnresolvedRecordLeftOut; TestMissingPackageLeftOut; TestRangeKeepsLowerBound; TestRangeGrammarValidated; TestScanFindingLeavesAdvisoryOut; TestPlanSendsOnlyGets; TestDigestStable
9. Approved publication
   - packages: internal/core/embargo (publish.go)
   - needs: 8
   - tests (fake gh recording each call): TestPublishWithoutApproveSendsNoPatch; TestPublishWrongDigestSendsNoPatch; TestCanaryFirstThenRest (call order); TestCanaryMismatchStops (no further PATCH); TestTriageStateAcceptedToDraftFirst; TestPartialRunReportsAndReplans; TestNoDraftLeftConfirmed; TestPublishedAtStamped
10. The CLI surface
   - packages: internal/surface/cli (embargo.go), helpgroups.go placement (release group, people block), internal/core/surface sentences.go, `go generate ./internal/surface/cli` to refresh .abcd/development/release/surface.json and docs/reference/cli/commands.md, `embargo` added to the DOGFOODING verb list in .abcd/rules.json
   - needs: 4, 5, 6, 7, 8, 9
   - --json on every sub-verb; text renders through termsafe; refusals exit 2 with the remedy
   - peers: `abcd peers` skips the embargo store worktree and says so in one line, so a private record's title never reaches its listing
   - tests: TestEmbargoSurfaceRefusals (each sub-verb); TestEmbargoJSONShapes; the sentence test; TestPeersSkipsEmbargoStore
11. The command page and the live run
   - packages: commands/embargo.md (`block:` frontmatter matching step 10's placement)
   - needs: 10
   - the page: the five sub-verbs; the approval rule for hosts, one approval per act for `close` and for `publish`; that the closing explanation is posted at the head of the advisory's description because GitHub offers no comment; what stays the person's act in this rung (fixing and landing, requesting a CVE, restoring redacted records); the check's paraphrase-and-count limit
   - live run on this repository, per evaluate-at-the-user-surface: intake into the store, triage one advisory, `close` without `--approve` on an advisory with a closing verdict (posting only with the person's approval in that session), check over the held batch range, publish plan without `--approve`; output kept in the local tier, never in the change
   - tests: the command-page gates (TestCommandPagesDeclareTheirBlock, TestCommandPagesMatchTheMenu, TestPluginPersonListEqualsTheCLIs) and `go run ./cmd/abcd lint docs`

## Delivery

After step 3, steps 4 to 6 and steps 7 to 9 touch disjoint files in the
package and can run as two lanes.

The change that finishes step 11 closes this spec (`abcd spec close`, with
the docs review for HEAD recorded first), carries `Delivers:
itd-2610112239022766` and, with iss-2610090703514921 moved to `resolved/`,
`Resolves: iss-2610090703514921`.

## Safety properties

Each is held by a named test above.

1. The embargo package never writes under the checkout's own
   `.abcd/work/issues/`: every write path is rooted at the store, and
   `VerifyStore` refuses a store that resolves inside the checkout
   (TestIntakeLeavesPublicLedgerUntouched, TestVerifyStoreRefusesStoreInsideCheckout).
2. Private records are never committed: the store is detached and every
   sub-verb refuses when the index tracks a record the state names
   (TestVerifyStoreRefusesBranchHead, TestVerifyStoreRefusesTrackedRecord).
3. abcd never prints advisory text into a public artefact: the check refuses
   ids, reserved CVE ids, titles and store paths, its findings mask the span,
   intake uses a neutral slug, and `peers` skips the store
   (step 7's tests, TestPeersSkipsEmbargoStore).
4. Closing refuses without explicit approval, and the approval binds to the
   text the person saw (TestCloseWithoutApproveSendsOnlyGet,
   TestCloseWrongDigestSendsNothing, TestCloseEditedDraftInvalidatesApproval).
5. Publication refuses without explicit approval, and the approval binds to
   the plan the person saw (TestPublishWithoutApproveSendsNoPatch,
   TestPublishWrongDigestSendsNoPatch).
6. A failed canary publishes nothing more (TestCanaryMismatchStops).
7. No repository-configured git program runs: every git call uses the
   isolated env, and history reads pass `--no-ext-diff --no-textconv`
   (TestCheckRunsNoRepoDiffDriver).
8. abcd holds no token; every remote call is the person's own `gh` against an
   explicit github.com host (TestRunnerNamesHostExplicitly).

## Progress

Fixes from the security review of 2026-10-12, each held by a named test:

- **Step 3, the store's modes.** The store's folder is 0700 and each private
  record 0600 (minted by intake, moved by triage); `Verify` refuses a folder
  or a named record with any group or other bit, naming the `chmod` that
  restores it (TestStoreRootOwnerOnly, TestVerifyStoreRefusesRootOpenToOthers,
  TestVerifyStoreRefusesRecordOpenToOthers).
- **Steps 3, 4 and 7, criteria 2 and 6: an advisory with no severity.** It
  was skipped before it entered the state, so its title and reserved CVE id
  never joined the deny set. The state (schema 2, reading schema 1) holds it
  as pending with no record, and the check refuses it
  (TestIntakeListsNullSeverityAndMintsNothing,
  TestCheckRefusesAdvisoryAwaitingSeverity,
  TestIntakeMintsPendingOnceSeverityIsSet, TestStateReadsSchemaOne).
- **Step 7, criterion 6: a closed advisory's title.** It stayed in the deny
  set forever; it now leaves it once `closed_at` is set, keeping the id and
  CVE id (TestCheckRefusesClosedUnpublishedGHSA,
  TestCheckKeepsClosedAdvisorysCVE, TestPlanLeavesOutAdvisoryNamingAnother).

## Out of scope

Later rungs, each a record of its own once this one ships: running fix lanes
from the store; landing the held branches together with redaction of the
fixed records and their restore after publication, with the three-way check; requesting a
CVE (`POST .../security-advisories/{ghsa_id}/cve`) at fix-ready; reporter
credits; the broken state (a leak, a second report or exploitation sends the
run straight to landing and publication); a push-to-publish timer (48 hours).
