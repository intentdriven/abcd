---
id: spc-2610031704338417
slug: when-abcd-sets-up-a-project-it-asks-the-owner-whether-abcd
intent: itd-2610031348087517
origin: researcher-authored
production_mode: hand-written
---
# when-abcd-sets-up-a-project-it-asks-the-owner-whether-abcd

## Summary

This spec delivers
[itd-2610031348087517](../../intents/planned/itd-2610031348087517-when-abcd-sets-up-a-project-it-asks-the-owner-whether-abcd.md)
(setup asks whether abcd keeps parts of the README current, and which). The
file name and slug are the ones minted on filing; the intent's title is the one
that describes the work.

When setup runs, it asks the owner whether abcd keeps parts of the project's
README.md current. On a yes it asks which parts, each as a tab with its current
setting, how to change it later and the exact lines it would write. abcd stores
the answer in `.abcd/config.json` and, on that and every later setup run,
derives each chosen part from the project on this computer and writes it inside
its own named marked place in README.md, with a hash of the text it wrote. A
part whose text no longer matches its hash was edited by hand: it is named and
left. A duplicated or unclosed mark refuses the whole write and names the line.
A part that cannot be derived is omitted with the reason. A project with no
README.md gets one holding only the chosen parts. Uninstall removes only the
parts whose text still matches its hash.

The settled records are designed to here, never reopened: the intent's
decisions 1 to 8, its confirmed criteria R1 to R8, and its five scope
conditions.

## Scope

In:

- `internal/core/ahoy`: the named marked parts, a sibling of `marker.go`'s one
  unnamed block; the derivation of each part; the badge-address check and the
  outbound scrub; the setup step that asks the question pair and writes the
  parts; the stored setting and its flag; uninstall's removal of the parts; the
  offer gap and the offer view detection carries.
- `internal/core/ahoy/prompt_help.go`: the canonical help for the question pair,
  keyed `readme` and `readme.<part>` as the status line's elements are keyed
  `statusline.<element>`.
- `internal/surface/cli`: the `--readme-parts` flag beside `--docs-target`
  (`cli.go`, line 3791), the terminal prompter's rendering of a part's current
  setting and example (`stdinPrompter.Prompt`, `cli.go`, line 4254), and the
  uninstall receipt's README line beside `marker removed:` (`cli.go`,
  line 3813); the regenerated `docs/reference/cli/commands.md`.
- Pages and record: `commands/ahoy.md` (relaying the pair through the host's
  question tool, Q1 alone and the parts as tabs); `docs/how-to/install.md`
  (beside the `--docs-target` paragraph, line 307); the brief's
  `04-surfaces/01-ahoy.md` (the gap table under "Gaps, and how the apply pass
  asks about them", line 480, and the `abcd ahoy install` and
  `abcd ahoy uninstall` appendix entries, lines 845 and 878).
- R7's and R8's dated receipts, in the local tier.

Builds on and reuses:

- itd-3's marked-block arrangement as `marker.go` implements it: the symlink
  refusal, the guarded read (`fsutil.ReadGuarded` under `maxAhoyFileBytes`,
  `store.go`, line 1059), the file lock (`withRewriteLock`, `rewritelock.go`,
  line 45), the line-ending recovery (`detectEOL`), the live-heading rule
  (`firstOutOfFenceH1`, reading `mdrecord.Mask`) and the frontmatter rule
  (`frontmatterRe`). Reused as functions, never copied.
- The origin remote reader (`originURL`, `store.go`, line 35, which already
  scrubs credentials) and the github.com parser (`parseGitHubRemote`,
  `remote.go`, line 184).
- The outbound primitive `scanner.ScrubOutbound`
  (`internal/adapter/scanner/outbound.go`, line 53), which today has no caller.
- The docs-lint text seam: `lint.LoadConfig`, `lint.NewTokenChecker`
  (`textlint.go`, line 28) and `TokenChecker.LintComposed` (line 74).
- The question type and its limits check (`question.Question`,
  `question.CheckLimits`) and the setup limits test
  (`TestEverySetupQuestionPassesTheLimits`), which live on the asking branch of
  spc-2610030944505997 and are not
  yet at this spec's base; step 4 waits on them.
- [spc-2610031156364295](../closed/spc-2610031156364295-evaluate-whether-claude-md-can-be-removed-safely-now-that.md)
  step 2, which narrows the conventions-file question to AGENTS.md or nothing;
  the README pair is asked after it (decision 7).

Out:

- The check that reports a stale or hand-edited part between setup runs and
  writes nothing: itd-2610031651058674 (decision 1). This spec adds no
  detection gap for a stale part, no lint rule, and no `abcd ahoy` line about
  one.
- A title, a description or any prose (decisions 3 and 6); a version written as
  text (decision 4).
- A README under another name or in another folder (scope condition 1).
- Any write between setup runs, on a schedule or at a release cut, and any
  commit: the owner commits the result (scope condition 4).
- Any network read during setup (scope condition 5); any visibility read from
  the forge (scope condition 2).
- The conventions block itself: `markerBlockRe`, `installMarkerFile`,
  `removeMarkerFile`, `markerFileHasBlock` and `StripMarkerBlock` are
  unchanged. A named fence never matches `markerBlockRe` (`marker.go`,
  line 30), whose pattern requires ` -->` straight after `ABCD`, so neither
  reading sees the other's blocks.

## Approach

### Named marked parts, a sibling of the one-block rule

`marker.go` keeps one unnamed block per file, compares it byte for byte with an
embedded template, and collapses several blocks into one
(`composeMarkerReplacement`, line 256). A README part's text differs per
project, so that comparison cannot tell abcd's text from the owner's. A new
file, `readme_marker.go`, holds the sibling rule: one named place per part,
each carrying the hash of the text abcd last wrote into it.

```markdown
<!-- BEGIN ABCD:licence sha256=<64 hex digits> -->
Licensed under the MIT License; see [LICENSE](LICENSE).
<!-- END ABCD:licence -->
```

The hash is SHA-256 over the body: the lines between the two fences, joined
with LF whatever the file's line ending, with no trailing newline. A fence line
counts only where `mdrecord.Mask` does not set `MaskFence`, so a README that
shows the fences inside a code example is never read as holding a part.

Parsing a README yields, per part name, one of these states:

| State | What the file holds | Setup does |
| --- | --- | --- |
| absent | no fence for the part | writes it, if chosen and derivable |
| current | hash matches the body, and the body equals the fresh derivation | nothing |
| stale | hash matches the body, and the body differs from the fresh derivation | rewrites the body and the hash |
| edited by hand | hash does not match the body | names the part and its line; leaves it |
| duplicated | two opening fences with one name | refuses the whole write; names both lines |
| unclosed | an opening fence with no closing fence, or a closing fence with no opening fence | refuses the whole write; names the line |

A duplicated or unclosed fence refuses every part's write, since the file's
structure is then uncertain; a part edited by hand leaves only that part, and
the other parts are still written. A part present but no longer chosen, or no
longer derivable, is removed when its hash matches and named when it does not.
A part is rewritten where it stands: an owner may move a marked place anywhere
in the file.

Placing a part that is absent:

- when other parts are present, beside the nearest one in the fixed order
  below, before or after it as the order says;
- else after a leading frontmatter block (`frontmatterRe`);
- else after the first live level-one heading (`firstOutOfFenceH1`);
- else at the top of the file. Unlike the conventions block, a part is never
  appended at the end, so `appendsInsideOpenSpan`'s refusal never applies.

Each part is separated from the text around it by one blank line, and the
file's own line ending is kept (`detectEOL`). The read and the write hold the
file's lock (`withRewriteLock`), a symlinked README.md is refused as
`installMarkerFile` refuses one, and a file over `maxAhoyFileBytes` is refused
by name. A run in which every part is current writes nothing, so the file's
bytes and modification time are unchanged (R1).

### The parts, how each is derived, and when it is omitted

The parts, in their fixed order:

| Part | Writes | Derived from | Omitted, with the reason said, when |
| --- | --- | --- | --- |
| `release-badge` | a badge image of the latest release, linked to the forge's latest-release page | the origin remote | the project is not declared public; the origin is not on github.com |
| `build-badge` | the forge's own workflow status badge, linked to the workflow's runs | the origin remote and `.github/workflows/` | not declared public; origin not on github.com; no workflow chosen (open question 2) |
| `abcd-badge` | a "managed with abcd" badge linked to abcd's public repository | nothing in the project | never omitted once chosen |
| `licence` | one line naming the licence and linking the file, or linking it unnamed | `LICENSE`, `LICENSE.md` or `LICENSE.txt` at the root | no such file |
| `links` | a list of links to the files found | `CHANGELOG.md`, `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, each looked for at the root, then `.github/`, then `docs/` | none of the four is found |
| `contents` | a list of links to the README's own sections | the README's live level-two and level-three headings outside every part | the README has no level-two heading |

The licence is named only from an `SPDX-License-Identifier:` line in the file's
first twenty lines, or from a forge-standard first line held in one table:
"MIT License" (MIT), "Apache License" followed by "Version 2.0" (Apache-2.0),
"BSD 2-Clause License", "BSD 3-Clause License", "ISC License" and "Mozilla
Public License Version 2.0". The GNU licences are not in the table, because
their text alone does not say whether "or any later version" applies; they are
named only from an identifier line. Any other LICENSE is linked without being
named (`See [LICENSE](LICENSE) for the licence terms.`). No licence text is
matched as a whole, so no dependency is added.

The `contents` anchors follow the forge's heading-slug rule (lower case,
punctuation other than hyphens dropped, spaces to hyphens, a repeated slug
numbered `-1`, `-2`), and a heading inside a part is never listed.

Derivation reads files and runs `git remote get-url origin` through
`originURL`; it imports no network package and calls nothing that does.
`readme_derive.go` takes a root directory and the stored settings and returns,
per part, either the body or the reason it is omitted.

### Three checks on every derived body

Before a body is written, it passes three checks in order, and a part failing
any one is omitted with the check's finding named:

1. **The badge-address check** (decision 7; R6). Every address in a badge part
   is parsed with `net/url`, used only as a parser. The scheme must be
   `https`. The host must be `github.com` for a link and for the build badge's
   image, and the badge-drawing host of open question 1 for the release and
   abcd badges' images. A query string is allowed only for the build badge, and
   only the keys the forge reads there (`branch`, `event`). Anything else is
   refused naming the piece: "build-badge refused: the address carries a query
   key the forge does not use: token". The derivation composes addresses from
   the origin remote, so a remote carrying a query string or a foreign host is
   what the check meets in practice.
2. **The outbound scrub.** The body goes through `scanner.ScrubOutbound` with
   the label `README.md part <name>`. Any finding refuses the part, naming the
   finding's rule: a part is never written with a masked span inside it. A
   scanner configuration that refuses to load refuses every part with its
   reason. The record makes no claim that `scanner.OutboundPolicy` covers badge
   addresses; the first check is what covers them.
3. **The project's own docs-lint.** Where the project keeps
   `.abcd/docs-lint.json` (README.md is one of the seeded roots), its banned
   tokens are compiled with `lint.NewTokenChecker` and the body linted with
   `LintComposed("README.md", body, false, nil)`. A blocker finding refuses the
   part naming the token's id; a warning is noted and the part written. A
   project with no such file skips this check.

### The question pair, in setup's fixed order

The pair is asked by a new step, `stepReadme`, in `readme_apply.go`. It runs
after `stepMarker` and before `stepSymlink`, so its questions come after the
category approvals, the configuration values (the conventions-file question
among them) and the house-style question, and before the status-line offer
(decision 7). The step asks only while `readme.parts` is absent from
`.abcd/config.json`, and only of a person at a terminal (`atTerminal`,
`identity_establish.go`, line 18), as the drain record is asked: off a terminal
a piped answer stream keeps the order it has today.

Q1 stands alone, keyed `readme`:

- `yes`: asks Q2.
- `no`: stores `readme.parts: []`, so the pair is not asked again; abcd's parts
  already in the file are removed as a narrowed set removes them.
- `later` (the default, and end of input): stores nothing; the pair is asked on
  the next setup run.

Q2 asks one on/off question per offered part, keyed `readme.<part>` in the
fixed order, which the host's question tool shows as tabs of at most four: two
calls for a project declared public (six parts), one call otherwise (four).
The defaults: `licence` and `links` on; `contents` off (the press release's "if
the owner wants one"); `release-badge` and `build-badge` on where offered;
`abcd-badge` off (decision 2). Every tab also offers "Decide later"; any
decide-later answer stores nothing, and the pair is asked again next time,
since the stored set is one answer.

Each part's question carries four things:

- what the part is (the static help in `promptHelp`, `prompt_help.go`,
  line 78, looked up through `HelpFor`, line 175, which gains the `readme.`
  prefix as it has `statusline.`);
- `Now:` the part's current setting (on, off or not set);
- `Change later:` the flag, `abcd ahoy install --readme-parts <list>|none`;
- an example: the exact lines it would write in this project, or the reason it
  writes none ("no LICENSE file").

The release and build badges are not asked about in a project not declared
public; Q2 says, in one line above its first tab, that it goes by the
declaration in abcd's settings and names it ("not declared public").

The current setting and the example are per run, so `PromptHelp` (line 27)
gains `Now` and `Example` fields, and an optional prompter interface beside
`TerminalPrompter`:

```go
type HelpedPrompter interface {
	Prompter
	PromptHelped(h PromptHelp, choices []string, def string) string
}
```

`stepReadme` asks through it when the prompter implements it, else through
`Prompt`. The CLI's `stdinPrompter` implements it by printing the two new
fields under the help it prints today; the question line is unchanged, so a
transcript lines up as before.

Through the plugin page the pair is relayed with the host's question tool.
Detection carries a `readme` view on `DetectionResult` (`ahoy.go`, line 90):
per part its name, whether it is offered, its `now`, its `example` or its
`reason`. The page asks Q1 alone, then the offered parts as tabs of at most
four, each with that view's lines, and passes the answer through
`--readme-parts`. Both routes build their words from the same core help, and
the setup limits test holds the pair to `question.CheckLimits` (step 4).

### Non-interactive runs and the stored answer

The setting is `readme.parts` in `.abcd/config.json`, an array of part names
written under the config lock as `stepConfigValues` writes its keys
(`apply.go`, line 534):

- absent: not yet answered, or answered "decide later";
- `[]`: answered no; never asked again;
- a list: the chosen parts, which every setup run then rewrites (decision 1).

The flag `--readme-parts` takes a comma-separated list of part names, or
`none`, which stores `[]`. An unknown name, or `none` beside a name, is refused
naming the known names, and nothing is written. The flag answers the pair in
any run, `--yes` included, and an explicit value that differs from the stored
one is echoed as a change (`readme.parts: licence -> licence,links`) and makes
an otherwise current repository do the work, as `overridesWouldChange` does
for the four configuration values.

While `readme.parts` is absent, detection raises an optional, resolvable gap,
`readme.offered`, in a new category `readme`. It joins `optionalGapIDs`
(`apply.go`, line 1789), so `--yes` and a run off a terminal report it under
`optional_skipped`, and the summary says: "The README question was not asked;
to answer it without being asked, pass --readme-parts
licence|links|contents|release-badge|build-badge|abcd-badge|none" (R7). The
category is kept out of the approval questions (open question 5): Q1 is the
consent.

A chosen set rewrites its parts on every run. So that a run in an otherwise
current repository still does it, `install`'s early return gains
`readmeWouldChange`, beside `attributionWouldChange` (`attribution_hook.go`,
line 110): it plans the parts without writing and reports whether any would
change. Detection itself raises no gap for a stale part; that report is
itd-2610031651058674's.

### A README created when missing

With no README.md and at least one part derivable, setup creates README.md
holding only the chosen parts, in the fixed order, with no title and no prose
(decision 5; R4). The owner's yes to the parts is the hand-over the principle
[the-users-directory-is-theirs](../../principles/the-users-directory-is-theirs.md)
asks for, and setup records `readme.created: true` beside the parts so the file
is one abcd can prove it wrote (open question 4). With no part derivable, no
file is created and the summary names each reason.

### Uninstall

`Uninstall` (`apply.go`, line 1611) gains a README pass after its conventions
loop (line 1621), whatever the stored setting says, as that loop ignores
`docs.target`:

- a duplicated or unclosed fence leaves the file untouched and names the line;
- every part whose hash matches its body is removed, with the blank lines
  setup added;
- every part edited by hand is left and named;
- a README.md that `readme.created` says abcd made, and that holds nothing but
  blank lines once its parts are removed, is deleted; any other file is kept,
  even when empty.

`UninstallReceipt` (`ahoy.go`, line 241) gains a `readme` field beside
`marker`: the parts removed, and each part left with its reason and line. The
text render reads "links removed; licence edited by hand, left" (R5). An
owner's README.md holding no part is never written, so its bytes are unchanged.

### The opt-in managed-with-abcd badge

The `abcd-badge` part is offered in every project, off by default, and is
written only when the owner turns it on (decision 2, which extends the
2026-09-23 ruling F by the owner's consent). Its image is a static badge
reading "managed with abcd", linked to abcd's public repository; it reads
nothing from the project, so it is never omitted once chosen. The fences of
every part name abcd too, as the conventions block's fences do, and they are
written only on the owner's yes.

## How each acceptance criterion is met

**R1.** `TestInstallWritesChosenReadmePartsOnce`, in `internal/core/ahoy`:
README.md holds `# Project` and a paragraph, LICENSE's first line is "MIT
License", and the stored set is `["licence"]`. After the first
`ahoy install`, one `licence` part naming MIT sits after the heading and every
other line is byte-identical; the second run's result lists no write, and the
file's bytes and modification time equal those after the first run.
`TestReadmePartStates` holds the six states of the table above.

**R2.** `TestInstallLeavesHandEditedOrBrokenReadme`, three cases: a licence
part whose body was edited, two `licence` opening fences, and an opening fence
with no closing fence. In each, README.md is byte-identical after the run and
the result's notes name the part, the reason and the line ("licence part
edited by hand at line 4; left as it is").

**R3.** `TestReadmeDerivationMakesNoNetworkCall`: a project with no LICENSE,
`repo.visibility` `private`, and the `licence`, `release-badge` and
`build-badge` parts chosen. `http.DefaultTransport` is replaced by a counting
round-tripper and `HTTPS_PROXY`, `HTTP_PROXY` and `ALL_PROXY` point at a
listener counting connections. None of the three parts is written, the notes
say "no LICENSE file" and "not declared public", and both counters read zero.

**R4.** `TestInstallCreatesReadmeWithOnlyChosenParts`: no README.md, a LICENSE
and a CHANGELOG.md, the stored set `["licence", "links"]`. README.md is created
holding the `licence` part then the `links` part and nothing else; its first
line is the licence part's opening fence.

**R5.** `TestUninstallRemovesOnlyCurrentReadmeParts`, two cases: an owner's
README.md with no part is byte-identical after `Uninstall`; a README.md with a
current `links` part and an edited `licence` part loses the `links` part only,
and the receipt's text reads "links removed; licence edited by hand, left".

**R6.** `TestBadgeAddressRefusesForeignQueryOrHost`: an origin remote whose
URL carries `?token=abc`, and one at a host that is not github.com. The
`build-badge` part is refused naming `token` in the first case and the host in
the second; README.md is not written in either. The check's table test also
holds the allowed `branch` key and every refused scheme.

**R7.** `TestInstallYesLeavesReadmeAndNamesFlag` asserts in a go test what the
receipt then records by hand: `ahoy install --yes` in a fresh repository with
no `--readme-parts` leaves README.md untouched (or absent), lists
`readme.offered` under `optional_skipped`, and the summary names
`--readme-parts`. The receipt,
`.abcd/.work.local/logs/readme-parts-yes-<yyyy-mm-dd>.md`, is taken with
`go run ./cmd/abcd ahoy install --yes` in a fresh scratch repository and records
the date, the abcd version, the summary quoted and the README.md state.

**R8.** The setup limits test (`TestEverySetupQuestionPassesTheLimits`) gains
Q1 and every Q2 tab, each built with a current setting, a change-later line
and an example, and holds them to `question.CheckLimits`: Q1 alone with its
decide-later option, Q2 in calls of at most four tabs, every tab within 24 rows
at 80 columns. `TestReadmeNoIsStored` answers no and asserts a second run asks
nothing; `TestAbcdBadgeOffByDefault` asserts the badge's default answer is off.
The receipt, `.abcd/.work.local/logs/readme-questions-<yyyy-mm-dd>.md`, is taken
through `/abcd:ahoy` in the harness's terminal view at 80 columns and records
the date, the harness version, a screenshot of Q1 and of each tab call, and the
second run asking nothing after a no.

## Open design questions

These are for the technical facilitator; the records do not settle them. Each
is designed to the marked option, and the reason is given beneath it.

1. **Where the release badge's image comes from.** The forge keeps the latest
   release live but draws no release badge of its own. (a) The shields.io
   release badge drawn from the forge's data, as abcd's own README shows one,
   linked to the forge's latest-release page; the badge-address check admits
   `img.shields.io` for the release and abcd badges' images and nothing else
   (designed to). (b) No image: a plain "Latest release" link to the forge's
   latest-release page, which the forge keeps live, and no third-party host.
   (c) No release part until the forge draws its own badge.
   - (a): it is the badge the press release promises, no version is written
     into the file (decision 4), and the one third-party host is named in a
     single allowed list the check holds.
2. **Which workflow the build badge shows.** (a) The only workflow file in
   `.github/workflows/`; where there are several, the one named `ci.yml` or
   `ci.yaml`; else the part is omitted naming the files found (designed to).
   (b) One badge per workflow. (c) A stored setting naming the workflow,
   asked in Q2.
   - (a): a badge wall is what the research advises against, and (c) adds a
     question the limits must hold for a case the rule covers.
3. **Which forge.** (a) github.com only, the one remote form abcd parses
   today (`parseGitHubRemote`); any other origin gets no badge part, with the
   reason (designed to). (b) A table of forges with each one's badge form.
   - (a): it adds no parser, and the refusal says why a badge is missing.
4. **An emptied README abcd created.** (a) Record `readme.created` when setup
   creates the file, and delete it at uninstall only when that record says so
   and nothing but blank lines remains (designed to). (b) Always leave the
   file. (c) Delete any file left empty.
   - (a): it removes only what abcd can prove it wrote; (c) could delete an
     empty file the owner made, and (b) leaves a file the owner never asked
     for.
5. **Whether the `readme` category gets an approval question.** (a) No: Q1 is
   the consent, and `resolveApproval` (`apply.go`, line 1911) leaves the
   category out of its questions as it leaves the drain record out off a
   terminal (designed to). (b) The status line's pattern: "Apply readme
   changes?" among the approvals, then Q1.
   - (a): one question fewer, and R8 asks that the first README question
     stand alone.
6. **One badge per marked place.** (a) Each badge is its own part, so each
   renders on its own line (designed to). (b) One `badges` part holding the
   chosen badges on one line.
   - (a): the press release keeps each part "inside its own marked place", and
     a badge the owner edits is then left without the others.

## Footprint

- packages: internal/core/ahoy, internal/surface/cli, commands/ahoy.md, docs/how-to, docs/reference, .abcd/development/brief/04-surfaces
- tests: TestReadmePartStates, TestReadmePartFenceRoundTrip, TestReadmePartFencesInsideCodeAreIgnored, TestReadmePartPlacement, TestReadmePartKeepsLineEndings, TestConventionsBlockIgnoresNamedFences, TestLicencePartNamesOnlyFromIdentifierOrFirstLine, TestLinksPartListsOnlyFilesThatExist, TestContentsPartListsLiveHeadings, TestBadgePartsNeedPublicAndForge, TestBadgeAddressRefusesForeignQueryOrHost, TestReadmePartRefusedByOutboundScrub, TestReadmePartRefusedByDocsLint, TestReadmeDerivationMakesNoNetworkCall, TestInstallWritesChosenReadmePartsOnce, TestInstallLeavesHandEditedOrBrokenReadme, TestInstallCreatesReadmeWithOnlyChosenParts, TestUninstallRemovesOnlyCurrentReadmeParts, TestInstallYesLeavesReadmeAndNamesFlag, TestReadmePartsFlagStoresTheAnswer, TestNarrowedReadmePartsRemoveDropped, TestReadmeOfferKeepsPipedAnswerOrder, TestReadmeQuestionPairOrder, TestReadmeNoIsStored, TestReadmeLaterStoresNothing, TestAbcdBadgeOffByDefault, TestReadmeOfferView, TestEverySetupQuestionPassesTheLimits (extended); the command reference's drift test after regeneration; docs-lint and record-lint clean; the two dated receipts in the local tier

## Steps

1. Named marked parts in README.md
   - criteria: the file halves of R1, R2 and R5
   - packages: internal/core/ahoy
   - tests: TestReadmePartStates, TestReadmePartFenceRoundTrip, TestReadmePartFencesInsideCodeAreIgnored, TestReadmePartPlacement, TestReadmePartKeepsLineEndings, TestConventionsBlockIgnoresNamedFences, each watched fail first; every existing marker test passes unchanged
2. Deriving the parts, and the three checks on each body
   - criteria: R3, R6, and the derivation halves of R1 and R4
   - packages: internal/core/ahoy
   - tests: TestLicencePartNamesOnlyFromIdentifierOrFirstLine, TestLinksPartListsOnlyFilesThatExist, TestContentsPartListsLiveHeadings, TestBadgePartsNeedPublicAndForge, TestBadgeAddressRefusesForeignQueryOrHost, TestReadmePartRefusedByOutboundScrub, TestReadmePartRefusedByDocsLint, TestReadmeDerivationMakesNoNetworkCall, each watched fail first
   - lands after step 1
3. Setup writes the stored parts, the flag answers the pair, and uninstall removes them
   - criteria: R1, R2, R4, R5, and the core half of R7
   - packages: internal/core/ahoy, internal/surface/cli, docs/reference
   - tests: TestInstallWritesChosenReadmePartsOnce, TestInstallLeavesHandEditedOrBrokenReadme, TestInstallCreatesReadmeWithOnlyChosenParts, TestUninstallRemovesOnlyCurrentReadmeParts, TestInstallYesLeavesReadmeAndNamesFlag, TestReadmePartsFlagStoresTheAnswer, TestNarrowedReadmePartsRemoveDropped, TestReadmeOfferKeepsPipedAnswerOrder, each watched fail first; the existing install and uninstall tests pass unchanged; the command reference regenerated with `go generate ./internal/surface/cli`
   - lands after step 2, and waits on spc-2610031156364295 step 2, which changes the same setup code (`stepConfigValues`, `stepMarker`, the docs-target flag) and settles the conventions-file question the pair follows
4. The question pair at a terminal and through the plugin page
   - criteria: R8, and the asked half of R7
   - packages: internal/core/ahoy, internal/surface/cli, commands/ahoy.md
   - tests: TestReadmeQuestionPairOrder, TestReadmeNoIsStored, TestReadmeLaterStoresNothing, TestAbcdBadgeOffByDefault, TestReadmeOfferView, each watched fail first; TestEverySetupQuestionPassesTheLimits extended with Q1 and every Q2 tab, owing nothing for them
   - lands after step 3, and waits on spc-2610030944505997's question type, limits check and setup limits test reaching the base
5. The how-to, the brief, the receipts and the close
   - criteria: R7 and R8's receipts; the intent's close
   - packages: docs/how-to, .abcd/development/brief/04-surfaces
   - tests: docs-lint and record-lint clean; both dated receipts taken at the step's branch tip before the pull request; a docs-fidelity review recorded for HEAD, then `abcd spec close spc-2610031704338417` (the intent already declares `impact: additive`) with a `Delivers: itd-2610031348087517` trailer
   - lands after step 4
