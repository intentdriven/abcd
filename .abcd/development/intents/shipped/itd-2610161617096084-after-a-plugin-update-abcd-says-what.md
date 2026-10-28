---
id: itd-2610161617096084
slug: after-a-plugin-update-abcd-says-what
spec_id: spc-2610170635097509
kind: bundle-member
suggested_kind: null
reclassification_history: []
builds_on: [itd-111, itd-105]
severity: minor
origin: researcher-authored
production_mode: hand-written
bundle: after-update
impact: additive
---

# After a plugin update, abcd says what changed for this repository

## Press Release

> After a plugin update, abcd says what changed in words a person can use. When the next session notices that the abcd plugin was updated, the agent's first reply gives one line per release saying what changed, with the full summary one command away: what waits on this repository, and anything that now needs the person to act. Alice, a solo founder, sees: abcd 0.16.0, live brief sessions are new and need setup here; ask for details. They know what changed without reading the whole changelog. The summary comes from a short digest each release carries inside abcd itself, never fetched from the internet.

## Why This Matters

_Proposed by the agent on 2026-10-16 from the code and the ledger, unconfirmed: the planning interview confirms or rewrites it._

After an update, abcd says only "abcd updated from X to Y" (`update.UpdatedFormat`,
shown once by whatever made the swap). What changed is in the changelog, and the
changelog is long: the 0.15.0 section is eight paragraph-length lines, three of
them marked "Breaking:" that ask a person to change what they type (the bare
`abcd decide` no longer mints an ADR; `--outbound-hook` and `--outbound-forge`
are refused). Nobody reads that at the start of a working session, so a person
learns of a breaking change when a command they relied on refuses, and of a new
feature not at all.

The release's own records already say what changed, release by release, each
line citing its record: `CHANGELOG.md`'s dated sections, `RELEASE.md`'s press
release and its archive under `.abcd/development/releases/`, and the shipped
intents' press releases. Today they sit on disk in the plugin root, because the
plugin root carries the repository. Not every line matters to every repository:
the outbound change matters only to a repository that opted into it, and a new
setup question matters only where it has not been answered.

## Mechanism

_Proposed by the agent on 2026-10-16, unconfirmed._

We expect people to act on a breaking change, and to try a new feature, within a
session of updating, because a short summary shown at the first use of the new
version, built from the release's own records, gives each change in one sentence
with where to read more. Today the same facts exist only in a changelog the
person has to think to open.

## Scope Conditions

_Proposed by the agent on 2026-10-16, unconfirmed._

- The summary's source travels inside the abcd binary: a digest the release cut <!-- cond: cond-2610170635095619 -->
  writes, one line per shipped record, embedded at build time. The plugin root
  cannot be relied on, because the curated payload itd-108 drafts
  (`.abcd/config/launch-payload.json`) carries none of `CHANGELOG.md`,
  `RELEASE.md` or `.abcd/`.
- Releases cut by `abcd launch ship`, whose changelog sections are derived from <!-- cond: cond-2610170635090370 -->
  records and cite record ids; sections rolled by hand (0.6.0 and earlier) are
  summarised as written, or named as not summarised.
- The version the repository or machine moved from is known on disk: the <!-- cond: cond-2610170635095099 -->
  repository's `setup_version` or the binary record's `previous_tag`.
- Each digest line is the record's own changelog line as the release cut <!-- cond: cond-2610170635096007 -->
  composed it (the line `launch ship` checks against the record id it cites),
  so the "one plain sentence" has one source and is never rewritten for the
  summary.

## Acceptance Criteria

Seeded by the agent on 2026-10-16 from the code and the two drafts, unconfirmed:
the planning interview walks each one.

- **Given** Alice, a solo founder, whose abcd moved from 0.15.0 to 0.16.0, **when**
  the agent's first reply after the update is written, **then** it carries one
  line for the release, naming what waits on this repository and anything that
  needs Alice to act, and one abcd command shows the full summary: each change
  as its digest line with the record id to read more, changes that need action
  first.
- **Given** an update that skipped releases (0.14.2 to 0.16.0), **when** the
  summary is shown, **then** it covers every release in between, and names the
  releases it covers.
- **Given** Alice's repository has setup questions waiting from this update,
  **when** the summary is shown, **then** it keeps what waits on this repository
  (questions not yet answered here, from ahoy's own detection) apart from what
  changed in abcd in general; the general part is shown once per computer and
  what waits here once per repository.
- **Given** a binary whose digest has no entry for a release in the range (a
  release cut before digests existed), **when** the summary is built, **then** it says which releases it could not describe and
  where their notes are published, and never fills the gap with a guess or
  presents a partial summary as complete.
- **Given** any session, **when** the summary is built, **then** nothing is
  fetched from the network: it reads only the digest inside the running binary,
  and the build passes the zero-network test harness.
- **Given** Alice has been shown the summary for 0.16.0, **when** later sessions
  start, **then** it is not shown again, and one abcd command shows it again on
  request.
- **Given** a session with no person to read it (a headless or scripted run),
  **when** the session starts, **then** the summary is not counted as seen, so the
  next session with a person shows it.
- **Given** Maya, an AI/agent researcher, working in a repository whose own
  `CHANGELOG.md` is not abcd's, **when** the summary is built, **then** its text
  comes only from the digest inside the running binary, never from the
  repository being worked in, and every line is sanitised as every other text
  abcd reads at run time.

## Decisions

The product thinker answered four of these on 2026-10-16; the agent decided the
rest on 2026-10-17 during an autonomous run, on the grounds given.

- **Shown on its own or on request:** one line in the first reply, the full
  summary on request (product thinker).
- **Who sees it:** the general news once per computer, what waits here once per
  repository (product thinker).
- **How long:** one line per release (product thinker).
- **What counts as "for this repository":** only what waits here, plus anything
  that needs the person to act; everything else is in the full summary (product
  thinker).
- **Which record is the source:** a digest the release cut writes, one line per
  shipped record taken from its changelog line, embedded in the binary, because
  the curated payload carries no changelog or records and the binary is the one
  thing every install has (agent, from the design review).
- **Which channel:** the session-start hook adds only a fixed line saying a
  summary waits; the agent runs an abcd verb that prints the summary and relays
  it in its first reply, which keeps the hook's fixed-text rule
  (iss-2608241115201044) (agent).

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-272fb9bcc5f0 -->
Fidelity review — receipt rcp-272fb9bcc5f0 (verifier abcd:intent-auditor claude-opus-5-5).

Provenance: abcd:intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:7ec1176939d01de26bafa5bd625689e47659921526101d0189699aabdc1cf754
Input attestations: diff:83f037c28..b80261f55 (b80261f55^1..b80261f55, the merge of PR #1091 feat/after-update delivering spc-2610170635097509; digest over git diff --binary of that range)@sha256:daba8f437518ee6933b1bbe4343c43cc56d382ae642fb6607a8e2acee59f15da; commit:b6c6f8ef6 (feat: ship the after-update bundle, Delivers: itd-2610161617096084)@-; tree:origin/main 2e03c61e454bf09ecda8cee0ae2f377157b522f7, read in the lowcpu-audits worktree at e850fe100, whose later commits add audit notes only; no file of the bundle (internal/core/afterupdate, internal/core/releasedigest, internal/core/release/digest.go, internal/core/ahoy/waiting.go, internal/surface/cli/update_news*.go, commands/update.md) changed after b80261f55@-;

Acceptance rollup: MET 4 · MET_WITH_CONCERNS 4 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: A constant session-start line sends the agent to update news, which prints one news line per release with its change count and the breaking record ids, and --full lists every digest line with its record ids, action first; concerns: what waits is a separate offer line that counts questions rather than naming them, not part of the release's line, and the claim that the first reply begins with these lines rests on a live run whose transcripts sit only in a session scratchpad (the first run restated the lines; only the re-run after the amended session line was byte for byte).
  evidence: internal/surface/cli/update_news.go:140 — "const updateNewsSessionLine = "abcd: an abcd update has news for this session. After the person's first message, " +"
  evidence: internal/core/afterupdate/afterupdate.go:490 — "line += fmt.Sprintf(", %d %s you to act (%s)", k, verb, actionIDs(r))"
  evidence: internal/core/afterupdate/afterupdate.go:541 — "return "abcd was updated from " + from + " to " + to + " since this repository was set up; " +"
  evidence: internal/core/afterupdate/afterupdate.go:582 — "out = append(out, mark+e.Line+" ("+strings.Join(ids, ", ")+")")"
  evidence: internal/core/afterupdate/afterupdate_test.go:409 — "" - needs you to act: The bare verb no longer mints a record. (itd-2610161616086564)","
  evidence: .abcd/development/specs/closed/spc-2610170635097509-after-update.md:690 — "transcripts and the store and config before and after are in the session scratchpad's `live-run/transcripts2/`"
- ac-2 — MET: Range walks the base-tag chain down from the running release to the start, the moment gives one line per release naming each tag newest first, a test from v0.14.2 gets lines for both v0.16.0 and v0.15.0, --full heads each release with the release it was cut after, and the embedded chain is unbroken from v0.15.1 to v0.7.0.
  evidence: internal/core/releasedigest/releasedigest.go:365 — "cur = r.BaseTag"
  evidence: internal/core/releasedigest/releasedigest_test.go:170 — "if got := strings.Join(tags(s.Releases), " "); got != "v0.16.0 v0.15.0 v0.14.2" {"
  evidence: internal/core/afterupdate/afterupdate_test.go:182 — ""abcd v0.15.0: 2 changes. Ask for details: abcd update news --full","
  evidence: internal/core/afterupdate/afterupdate.go:570 — "head += " (after " + r.BaseTag + ")""
  evidence: internal/core/releasedigest/digest.json:442 — ""base_tag": "v0.14.2","
- ac-3 — MET: The news lines come from the digest and the offer from ahoy.Waiting over the session's own detection, composed as separate lines and, in --full, under a separate heading; news-seen in the user home marks the news once per computer, an exclusive answer file keyed on the root commit and release marks the offer once per repository, and a test shows a second repository on the same computer gets its offer but not the news.
  evidence: internal/core/ahoy/waiting.go:82 — "for _, q := range armedQuestions(cwd, det) {"
  evidence: internal/core/afterupdate/afterupdate.go:9 — "~/.abcd.noindex/after-update/< root-sha>/< tag>.json this repository's answer for one release"
  evidence: internal/core/afterupdate/afterupdate.go:10 — "~/.abcd.noindex/after-update/news-seen the newest release whose news this computer was shown"
  evidence: internal/core/afterupdate/afterupdate_test.go:236 — "// A second repository on this computer: its own offer, but not the news."
  evidence: internal/core/afterupdate/afterupdate.go:589 — "out = append(out, "Waiting in this repository:")"
- ac-4 — MET_WITH_CONCERNS: A release the chain reaches but the digest lacks stops the walk and sets Gap and Partial, rendered as a line saying abcd cannot describe it, where its notes are published and that the summary is partial, and hand-rolled sections are never described; concern: by the step 2 amendment the gap names a span ('the releases after X through Y') with a notes address for its top release only plus the release list, not each release it could not describe.
  evidence: internal/core/releasedigest/releasedigest.go:351 — "s.Gap = &Gap{After: from, Through: cur, NotesURL: notesURL(cur), ReleasesURL: releasesURL()}"
  evidence: internal/core/afterupdate/afterupdate.go:504 — "what = "the abcd releases after " + g.After + " through " + g.Through"
  evidence: internal/core/afterupdate/afterupdate.go:506 — "return "abcd cannot describe " + what + "; its notes are at " + g.NotesURL +"
  evidence: internal/core/afterupdate/afterupdate_test.go:199 — "gap := "abcd cannot describe the abcd releases after v0.13.0 through v0.14.1; its notes are at " +"
  evidence: .abcd/development/specs/closed/spc-2610170635097509-after-update.md:222 — "**Amended 2026-10-17 by the agent, building step 2: a gap names a span,"
  evidence: internal/core/releasedigest/seed_test.go:103 — "derived := !launch.CoreGreater(first, v)"
- ac-5 — MET: The summary's one source is the digest go:embed carries, which Range reads through embeddedDigest, and TestUpdateNewsTouchesNoNetwork runs session start and every form of the verb under the existing counting transport and release-fetcher seam, failing on any request.
  evidence: internal/core/releasedigest/releasedigest.go:74 — "//go:embed digest.json"
  evidence: internal/core/releasedigest/releasedigest.go:169 — "d, err := embeddedDigest()"
  evidence: internal/surface/cli/hook_session_start_update_news_test.go:118 — "func TestUpdateNewsTouchesNoNetwork(t *testing.T) {"
  evidence: internal/surface/cli/hook_session_start_update_news_test.go:138 — "if n := atomic.LoadInt32(&hits); n != 0 {"
- ac-6 — MET_WITH_CONCERNS: news-seen, raised by --seen or --answer, starts the next moment's range after the release shown, so Pending reports no news and the hook prints no line, and --full starts at previous_tag, never news-seen, so it shows the summary again on request and writes nothing; concern: 'shown' is recorded only when the agent runs --seen or --answer after the person's reply, as commands/update.md instructs, so a relay without that run shows the news again and nothing in the binary ties the mark to the showing itself.
  evidence: internal/core/afterupdate/afterupdate.go:309 — "func() (string, bool) { return seen, seen != "" },"
  evidence: internal/surface/cli/hook_session_start_update_news_test.go:100 — "nothing is pending, yet the line printed"
  evidence: internal/surface/cli/update_news_test.go:178 — "if out, _, code := runUpdateNews(); code != 0 || out != "" {"
  evidence: internal/core/afterupdate/afterupdate_test.go:426 — "if err != nil || !slices.Equal(again.Lines, f.Lines) {"
  evidence: commands/update.md:232 — "`"${CLAUDE_PLUGIN_ROOT}/abcd" update news --seen` once the person has"
- ac-7 — MET_WITH_CONCERNS: The hook, bare update news and --full write nothing (tests assert the store is not created and the trees are unchanged), only --answer and --seen write, and the command page tells a headless agent to run neither; concern: the binary cannot tell a headless run from a person's, so 'not counted as seen' holds only while a headless agent obeys the page, a limit the spec states.
  evidence: internal/core/afterupdate/afterupdate.go:14 — "raised. Only Record and MarkSeen write. Pending, Moment and Full read, so a"
  evidence: internal/surface/cli/hook_session_start_update_news_test.go:51 — "t.Errorf("the hook wrote the after-update store (err %v)", err)"
  evidence: internal/surface/cli/update_news_test.go:116 — "if after := pathsUnder(t, home) + pathsUnder(t, repo); after != before {"
  evidence: commands/update.md:235 — "run neither `--answer` nor `--seen`: the binary cannot tell such a run from a"
  evidence: .abcd/development/specs/closed/spc-2610170635097509-after-update.md:463 — "- An agent that runs `--seen` or `--answer` without a person (against the"
- ac-8 — MET: Change lines come only from the embedded digest, each re-sanitised with termsafe on parse and again on print, versions pass ParseReleaseTag and waiting titles are core's words, and a CLI test plants a repository CHANGELOG.md carrying a directive and asserts neither the bare moment nor --full prints any of it.
  evidence: internal/core/releasedigest/releasedigest.go:243 — "line := termsafe.Sanitize(e.Line)"
  evidence: internal/surface/cli/update_news.go:174 — "fmt.Fprintln(w, termsafe.Sanitize(l))"
  evidence: internal/surface/cli/update_news_test.go:125 — "func TestUpdateNewsPrintsNoWordOfTheRepositorysChangelog(t *testing.T) {"
  evidence: internal/surface/cli/update_news_test.go:143 — "if strings.Contains(out, "curl") || strings.Contains(out, "example.invalid") {"
  evidence: internal/core/releasedigest/releasedigest_test.go:136 — "func TestParseSanitisesEveryLine(t *testing.T) {"

Gap audit:
- honoured:
  - After an update, the next session's first reply opens with one line per release, with the full summary one command away
    evidence: internal/surface/cli/update_news.go:140 — "const updateNewsSessionLine = "abcd: an abcd update has news for this session. After the person's first message, " +"
    evidence: internal/surface/cli/cli.go:2262 — "notices = append(notices, updateNewsSessionStart(cmd.OutOrStdout(), cwd, det)...)"
    evidence: internal/core/afterupdate/afterupdate.go:492 — "out = append(out, line+". Ask for details: "+FullCommand)"
  - Anything that needs the person to act is named on its release's line and listed first in the full summary
    evidence: internal/core/afterupdate/afterupdate.go:490 — "line += fmt.Sprintf(", %d %s you to act (%s)", k, verb, actionIDs(r))"
    evidence: internal/core/releasedigest/releasedigest.go:372 — "func actionFirst(in []Entry) []Entry {"
    evidence: internal/core/afterupdate/afterupdate.go:580 — "mark = " - needs you to act: ""
  - What waits on this repository is shown apart from the general news, from ahoy's own detection
    evidence: internal/core/ahoy/waiting.go:82 — "for _, q := range armedQuestions(cwd, det) {"
    evidence: internal/core/afterupdate/afterupdate.go:541 — "return "abcd was updated from " + from + " to " + to + " since this repository was set up; " +"
  - The summary comes from a digest carried inside abcd itself, appended by the release cut where config declares it, and never fetched from the internet
    evidence: internal/core/releasedigest/releasedigest.go:74 — "//go:embed digest.json"
    evidence: internal/core/release/digest.go:143 — "rel.Entries = append(rel.Entries, releasedigest.Entry{Line: e.Text, Records: recs})"
    evidence: .abcd/config.json:19 — ""digest": "internal/core/releasedigest/digest.json""
    evidence: internal/surface/cli/hook_session_start_update_news_test.go:138 — "if n := atomic.LoadInt32(&hits); n != 0 {"
- diverged:
  - The one line per release says what changed (the press release's 'live brief sessions are new and need setup here'): delivered, it says how many changes the release made and which record ids need action; what changed is only in --full, as the release's CHANGELOG lines
    evidence: internal/core/afterupdate/afterupdate.go:484 — "line := fmt.Sprintf("abcd %s: %s", r.Tag, plural(n, "change", "changes"))"
    evidence: internal/core/afterupdate/afterupdate_test.go:139 — "aliceNews = "abcd v0.16.0: 9 changes, 1 needs you to act (itd-2610161616086564). Ask for details: abcd update news --full""
  - The release's line names what waits here: delivered as a second, separate offer line that counts setup questions and changed defaults, with their titles only in --full
    evidence: internal/core/afterupdate/afterupdate.go:541 — "return "abcd was updated from " + from + " to " + to + " since this repository was set up; " +"
    evidence: internal/core/afterupdate/afterupdate_test.go:140 — "aliceOffer = "abcd was updated from v0.15.0 to v0.16.0 since this repository was set up; 2 setup questions wait here. Run setup now? (yes / not now)""
  - The releases the digest cannot describe are each named: delivered as a span ('the releases after X through Y') with the top release's notes address and the release list
    evidence: internal/core/afterupdate/afterupdate.go:504 — "what = "the abcd releases after " + g.After + " through " + g.Through"
    evidence: .abcd/development/specs/closed/spc-2610170635097509-after-update.md:222 — "**Amended 2026-10-17 by the agent, building step 2: a gap names a span,"
  - Shown once, and never counted as seen in a headless run: both rest on the agent running --seen or --answer only with a person's reply, as the command page instructs; the binary cannot tell
    evidence: commands/update.md:235 — "run neither `--answer` nor `--seen`: the binary cannot tell such a run from a"
    evidence: .abcd/development/specs/closed/spc-2610170635097509-after-update.md:463 — "- An agent that runs `--seen` or `--answer` without a person (against the"
  - The line appears when a session notices an update: where none of news-seen, previous_tag or setup_version is on disk, the range is the running release alone, so a computer's first session on a release in a repository not yet set up is told an update has news with no update having happened
    evidence: internal/core/afterupdate/afterupdate.go:298 — "// repository's setup version, else it is the running release alone."
    evidence: internal/core/releasedigest/releasedigest_test.go:253 — "func TestRangeWithNoStartIsTheRunningReleaseAlone(t *testing.T) {"
  - The next session notices the update on every host: a host that runs no session-start hook gets no line, and the summary is reached only by asking for abcd update news
    evidence: .abcd/development/specs/closed/spc-2610170635097509-after-update.md:467 — "- A host without hooks gets no line; `abcd ahoy` lists the waiting setup and"
  - Each release carries a digest the release cut wrote: every release described today (v0.7.0 to v0.15.1) was copied in by a one-off seed script or, for v0.15.1, a hand commit after its cut; no real cut has yet written the digest, and the cut's write is proven by internal/core/release tests alone
    evidence: 6addede52 — "The v0.15.1 cut ran on main before the cut wrote the digest"
    evidence: .abcd/development/specs/closed/spc-2610170635097509-after-update.md:240 — "once and held by tests, not by a kept generator.** A one-off script copied"
    evidence: internal/core/release/digest_test.go:73 — "func TestCutWritesTheDigestBeforeTheHeading(t *testing.T) {"
- missing:
  - A durable proof that a live host agent's first reply begins with the printed lines word for word: the spec's live run cites transcripts only in a session scratchpad's live-run/transcripts/ and live-run/transcripts2/, none of which is in the repository or in a durable local tier, and the first run's reply restated the lines
    evidence: .abcd/development/specs/closed/spc-2610170635097509-after-update.md:690 — "transcripts and the store and config before and after are in the session scratchpad's `live-run/transcripts2/`"
    evidence: .abcd/development/specs/closed/spc-2610170635097509-after-update.md:689 — "(restated, not verbatim)"
    evidence: .abcd/development/specs/closed/spc-2610170635097509-after-update.md:554 — "previous release's binary, and a live host session's first reply is captured"

Scope-condition dispositions:
- cond-2610170635095619 — narrowed: The source does travel in the binary (go:embed of internal/core/releasedigest/digest.json, which the cut appends to where config declares release.digest), and the curated payload lists none of CHANGELOG.md, RELEASE.md or .abcd/; but entries are one per changelog line, not one per shipped record, and no release cut has yet written the file.
  narrowing: holds as an embedded digest of one entry per changelog line (a line citing several records is stored once with every id, and a record with no changelog line has no entry), not one line per shipped record; and every release it describes today, v0.7.0 to v0.15.1, was copied in by a one-off seed script or a hand commit (6addede52), so the cut's own write is exercised only by internal/core/release tests until the next cut
  evidence: internal/core/releasedigest/releasedigest.go:74 — "//go:embed digest.json"
  evidence: internal/core/release/digest.go:143 — "rel.Entries = append(rel.Entries, releasedigest.Entry{Line: e.Text, Records: recs})"
  evidence: .abcd/development/specs/closed/spc-2610170635097509-after-update.md:110 — "**Each digest entry is one changelog line, with every record it cites.** A"
  evidence: .abcd/config/launch-payload.json:9 — ""README.md","
  evidence: 6addede52 — "The v0.15.1 cut ran on main before the cut wrote the digest"
- cond-2610170635090370 — narrowed: Derived sections are described and hand-rolled ones named as not described, never summarised as written; but the hand-rolled sections run through v0.6.9, not 0.6.0, so the digest describes v0.7.0 on.
  narrowing: holds for releases from v0.7.0 on, whose CHANGELOG sections cite record ids; v0.6.1 to v0.6.9 were rolled by hand too, and with 0.6.0 and earlier are only named as not described (as a span with its top release's notes address), never summarised as written
  evidence: internal/core/releasedigest/seed_test.go:20 — "const firstDescribed = "v0.7.0""
  evidence: internal/core/releasedigest/digest.json:7545 — ""base_tag": "v0.6.9","
  evidence: CHANGELOG.md:821 — "## [0.6.9] - 2026-08-29"
  evidence: .abcd/development/specs/closed/spc-2610170635097509-after-update.md:126 — "hand (v0.6.9 and earlier) are named as not described."
- cond-2610170635095099 — survived: Both named sources are read from disk and passed through ParseReleaseTag, setup_version through ahoy.SetupVersion and the binary record's previous_tag through ahoy.PreviousReleaseTag, each with its own test; the delivery puts news-seen ahead of them and falls back to the running release alone where none is known.
  evidence: internal/core/ahoy/waiting.go:199 — "func SetupVersion(cwd string) (string, bool) {"
  evidence: internal/core/ahoy/waiting.go:227 — "sv, err := launch.ParseReleaseTag(metaField(meta, "previous_tag"))"
  evidence: internal/core/afterupdate/afterupdate.go:309 — "func() (string, bool) { return seen, seen != "" },"
  evidence: internal/core/ahoy/after_update_reads_test.go:52 — "func TestPreviousReleaseTagReadsWithoutClaiming(t *testing.T) {"
- cond-2610170635096007 — survived: The cut stores each accepted changelog entry's composed text as the digest line, Append refuses a line sanitising would change rather than storing it altered, a test holds every embedded line equal to its CHANGELOG line, and --full prints the line unchanged with its record ids.
  evidence: internal/core/release/digest.go:143 — "rel.Entries = append(rel.Entries, releasedigest.Entry{Line: e.Text, Records: recs})"
  evidence: internal/core/releasedigest/releasedigest.go:309 — "if termsafe.Sanitize(e.Line) != e.Line {"
  evidence: internal/core/releasedigest/seed_test.go:94 — "if strings.Join(got, "\n") != strings.Join(sections[i].lines, "\n") {"
  evidence: internal/core/afterupdate/afterupdate.go:582 — "out = append(out, mark+e.Line+" ("+strings.Join(ids, ", ")+")")"
<!-- abcd-review-end receipt=rcp-272fb9bcc5f0 -->

## Grounds

- pursued: we expect people to act on a change that needs them within a session of updating, because one line per release in the first reply names it; a breaking change first noticed through a refused command after the summary was shown would show it wrong
