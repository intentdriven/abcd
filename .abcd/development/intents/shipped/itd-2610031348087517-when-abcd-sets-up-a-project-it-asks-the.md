---
id: itd-2610031348087517
slug: when-abcd-sets-up-a-project-it-asks-the
spec_id: spc-2610031704338417
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
refines: [itd-3]
related_intents: [itd-2610030814013772]
related_issues: [iss-2609110944498549]
severity: minor
origin: researcher-authored
production_mode: hand-written
impact: additive
---

# Setup asks whether abcd keeps parts of the README current, and which

## Press Release

> When abcd sets up a project, it asks the owner whether abcd should keep parts of the project's README.md current or leave the README entirely to them. If the owner says yes, abcd offers the parts it can read from the project itself: a line naming the licence; links to files the project has, such as its changelog, its contributing guide and its security policy; and, if the owner wants one, a table of contents. For a project the owner has declared public, it also offers a badge showing the latest release, which the forge keeps live so no version number is written into the file, and a badge showing whether the build passes. A badge saying the project is managed with abcd is offered as well, and stays off unless the owner turns it on. abcd keeps only the parts the owner picks, each inside its own marked place, and rewrites them each time setup runs. It never writes the title and never touches the owner's own words: a part the owner has edited by hand is reported and left as it is. A project with no README gets one holding only the chosen parts. A separate check, which reports a part gone stale between setup runs and writes nothing, is its own piece of work.

_Proposed by the facilitator from decisions 1 to 7; confirmed as written by the product thinker at the planning interview, 2026-10-03 (decision 8)._

Previous wording (superseded by decisions 1 to 7):

> When abcd sets up a project, it asks the owner whether abcd should keep parts of the project's README.md current or leave the README entirely to them. If the owner says yes, abcd asks which parts it keeps, such as the title and the badges, including a badge saying abcd manages the project, and it keeps only those parts, inside marked places, never rewriting the owner's own text.

> _Proposed by the facilitator on filing (2026-10-03) from the product thinker's request, quoted verbatim: "when running /abcd:ahoy, ask the repo owner whether they want abcd to manage the repo's main README.md, or whether they want them to maintain it. If the answer is yes, ask what elements abcd should maintain (run SOTA), incl." and, asked what "incl." introduced: "incl. title, badges (incl. an abcd-managed badge) etc."; to be confirmed at the planning interview._

## Why This Matters

_Facilitator-written._

A README goes stale in the parts that can be read from the repository itself: the version, the licence, the build status, the links to other files. abcd already keeps one marked block current in a project's conventions file where the owner chose one (the default writes none, iss-2609110944498549; under itd-2610030814013772 that file is AGENTS.md alone); this extends the same arrangement to the README, but only where the owner asks and only for the parts they choose, because a project's files are the owner's (the principle the-users-directory-is-theirs: abcd writes only where it is handed a place, and removes only what it can prove it wrote).

Typed links: refines itd-3's marked-block arrangement (shipped), which setup already plants in the conventions file; the question joins setup's fixed question order after the conventions-file question (decision 7). The check that reports a stale or hand-edited part is itd-2610031651058674, which builds on this draft (decision 1). A state-of-the-art pass on which README elements tools keep current, and how they mark them, was started on 2026-10-03 at the product thinker's request; its report feeds this draft's reviews.

## Decisions

1. 2026-10-03, the product thinker at the planning interview, asked what keeping the parts current means (a rewrite each time setup runs; that plus a separate check reporting stale parts; decide later): both. Setup rewrites the chosen parts each time it runs, and a separate check that reports a stale or hand-edited part, writing nothing, is filed as its own draft that builds on this one.
2. 2026-10-03, the product thinker, told that a standing ruling (2026-09-23, ruling F) allows one mention of abcd in a project abcd sets up and that a visible badge would be a second: the badge is its own part, opt-in and off by default. This extends ruling F by the owner's explicit consent; it is recorded as the product thinker's decision, not the facilitator's.
3. 2026-10-03, the product thinker: the title is off the list; abcd never writes a title (nothing in a project derives one reliably).
4. 2026-10-03, the product thinker: the version appears as the forge's live release badge, never as a version written into the file (adr-19 keeps version numbers out of committed files); public projects only.
5. 2026-10-03, the product thinker, asked what happens in a project with no README (write nothing until the owner makes one; create a README holding only the chosen parts; decide later): abcd creates a README holding only the chosen parts. The facilitator records the tension with the-users-directory-is-theirs for the spec: the owner's yes to the parts is the hand-over, and the file is one abcd can prove it wrote.
6. 2026-10-03, decided without a question (both reviews and the principle agree): the owner's prose is never touched; a hand-edited part is reported and left; a duplicated or unclosed marker refuses the write and names the line.
7. 2026-10-03, decided without a question (the technical facilitator's): the question joins setup's fixed order after the conventions-file question; a non-interactive setup declines it and names the flag; a no is stored so it is not asked again; the build-status badge is offered only where the owner declared the project public; a badge address with a foreign query string or host is refused by name.
8. 2026-10-03, the product thinker: the revised press release confirmed as written.

## Open Questions

None open for the product thinker: the interview of 2026-10-03 answered them (decisions 1 to 5). Owed at planning, for the product thinker to confirm: the revised press release, the scope conditions, and each criterion walked with its example.

Left for the technical facilitator's spec: the stored setting's name and the flag's spelling, the marked place's exact form, and the generated text passing the project's own documentation lint where it keeps those rules.

## Mechanism

We expect the chosen README parts to stop going stale because setup rewrites them from the project itself, and the check reports any that drift between setups; a chosen part found stale after a setup run shows the claim wrong.

_Proposed by the facilitator; confirmed by the product thinker, 2026-10-03._

## Scope Conditions

- A README named README.md at the root of the project; a README under another name or in another folder is out of scope and is left alone. <!-- cond: cond-2610031704331158 -->
- Public means what the owner declared in abcd's settings: abcd reads no visibility from the forge, so the release and build-status badges are offered only under that declaration, and a project not declared public gets neither, with the reason said (decisions 4 and 7). <!-- cond: cond-2610031704332577 -->
- The licence is named only from a licence identifier line or a forge-standard first line in the LICENSE file; any other LICENSE is linked without being named, and a project with no LICENSE gets no licence part. <!-- cond: cond-2610031704332960 -->
- Setup run by a person, with the question answered or the flag given; nothing rewrites the README between setup runs or on a schedule, and committing the result is the person's own act (decision 1). <!-- cond: cond-2610031704338772 -->
- Every part is read from the project on this computer; setup asks the network nothing for it, and the live badges are drawn by the forge when someone reads the page. <!-- cond: cond-2610031704334415 -->

_Proposed by the facilitator from decisions 1 to 7 and the design review; confirmed by the product thinker, 2026-10-03._

## Acceptance Criteria

_Proposed by the facilitator from decisions 1 to 7 and the design review's criteria that survive them; each is unconfirmed until walked with its addressee. Review criterion C3 is rewritten from a version read from tags to the public declaration (decision 4); C7 joins the README created when missing (decision 5); C5, the drift lint, belongs to itd-2610031651058674 (decision 1) and is left out._

- R1 (technical facilitator; CONFIRMED 2026-10-03) Given a README.md holding the owner's heading and prose and no abcd part, with the licence the only part chosen and a LICENSE whose first line is "MIT License", when setup (`abcd ahoy install`) runs twice, then after the first run one marked licence part naming MIT sits after the first heading, the owner's lines are unchanged, and the second run writes nothing (example: the file's bytes and modification time are the same after the second run as after the first); a go test asserts it.
- R2 (technical facilitator; CONFIRMED 2026-10-03) Given a licence part whose text was edited by hand, or two parts both marked as the licence, or an opening mark with no closing mark, when setup runs, then README.md is byte-identical and the summary names the part, the reason and the line (example: "licence part edited by hand at line 4; left as it is"); a go test asserts all three.
- R3 (product thinker; CONFIRMED 2026-10-03) Given a project with no LICENSE file and not declared public, when setup derives the chosen licence, release-badge and build-status parts, then none of the three is written, the summary says "no LICENSE file" and "not declared public", and no network call is made (example: a test harness that counts network requests counts none); a go test asserts it.
- R4 (product thinker; CONFIRMED 2026-10-03) Given a project with no README.md and the licence and links parts chosen, when setup runs, then README.md is created holding only those two parts in their marked places, with no title and no prose (example: the file begins with the licence part's opening mark); a go test asserts it.
- R5 (product thinker; CONFIRMED 2026-10-03) Given an owner's README.md holding no abcd part, when abcd is uninstalled, then the file is byte-identical; given one current part and one edited by hand, then only the current part is removed and the edited one is named as left (example: the summary reads "links removed; licence edited by hand, left"); a go test asserts both.
- R6 (product thinker; CONFIRMED 2026-10-03) Given a build-status badge address carrying a query string the forge does not use, or a host that is not the forge, when the part is derived, then the part is refused with the offending piece named and nothing is written (example: an address ending `?token=abc` is refused naming `token`); a go test asserts it.
- R7 (product thinker; CONFIRMED 2026-10-03) Given a fresh scratch repository, when setup runs with `--yes` and no README flag, then README.md is untouched (or absent, if it was absent) and the summary says the README question was not asked and names the flag; a dated receipt in the local tier records the date and the abcd version.
- R8 (product thinker; CONFIRMED 2026-10-03) Given the setup interview in the harness's terminal view at 80 columns, when the README questions are asked, then the first question stands alone with "decide later", the parts appear as tabs of at most four, each with its current value, how to change it later and an example of what it writes, the managed-with-abcd badge shows as off, and a no answered here is not asked again on the next setup run; a dated receipt with a screenshot, the date and the harness version records it.

Impact expected: additive. Setup gains one question, asked after the conventions-file question, and writes the README only on the owner's yes; nothing setup does today changes meaning.


## Review findings (design and record discipline, 2026-10-03)

From reports/review-readme-design.md and reports/review-readme-records.md in the local tier, and the research report sota-readme.md beside them. The interview of the same day settled the findings that were the product thinker's to rule; those stand as decisions 1 to 7 and are not reopened here.

Applied:

- Design 3: the build-status badge is offered only where the owner declared the project public, and the question says it goes by that declaration (decision 7; a scope condition).
- Design 4: a badge address with a foreign query string or host is refused by name (decision 7; criterion R6); the record does not claim the outbound policy covers it.
- Design 5: the licence is named only from an identifier line or a forge-standard first line, otherwise linked (a scope condition; criterion R1).
- Design 6: one marked place per part; a hand edit is reported and left, and a duplicated or unclosed mark refuses the write and names the line (decision 6; criteria R1, R2 and R5).
- Design 8: setup rewrites the chosen parts on each run, and no release cut writes the README (decision 1; a scope condition).
- Design 10: the first question stands alone with "decide later", the parts come as tabs, a non-interactive setup declines and names the flag, and a no is stored (decision 7; criteria R7 and R8); the setting's name and the flag's spelling are left to the spec.
- Design 11 and records 5: the conventions file is AGENTS.md alone, as Why This Matters already said; "the command list" leaves the list of parts that go stale, because the research rejects it.
- Records 1: the drift check is its own draft, itd-2610031651058674, which builds on this one (decision 1).
- Records 2: the front matter's typed links stand; the prose now refines itd-3's marked-block arrangement rather than an interview itd-3 never had.
- Records 3: the badge's relation to ruling F is the product thinker's to rule, and decision 2 rules it; the credit-only rule is no longer cited.
- Records 6: Why This Matters is marked facilitator-written, and the press release is re-proposed after the interview for the product thinker to confirm.

Overtaken:

- Design 1 and records 4 (the title): decision 3; abcd never writes a title.
- Design 2 and records 4 (the version from tags): decision 4; the version is the forge's live release badge, public projects only, and no version is written into the file.
- Design 7 (no README means nothing written): decision 5; abcd creates a README holding only the chosen parts.
- Design 9 (the badge framed as taste): decision 2; the badge is its own part, off by default.
- Design criterion C5 (the drift lint): decision 1; it belongs to itd-2610031651058674.
- The records review's disagreements with the design review (findings 1, 7, 8 and 9): settled by decisions 1, 2, 3 and 5.
- The records review's interview questions Q1 to Q8: answered by decisions 1 to 7, and the Mechanism confirmed.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-5b7cb4b81566 -->
Fidelity review — receipt rcp-5b7cb4b81566 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:5bacf39e3b1d18cc1236db69cb6b11bf098a21e4d8a9b7c6cc30bfef61ca3ddb
Input attestations: diff:a09dd42b6de9a6556d5ddc49b8810f65b5064b61..a639bb8927714f1ca7bbff7e4a8e8ef6b11de0af@sha256:a5f23c9502f339c6282731f4d964baea306d5e428a148f6b14072d69b4d1e2aa; diff:3d171421a699f552797aa7c7135ea64b646568fd..cb1258cdfd7fb22fa0e4fc7f1bfd195fd141bd3d@sha256:6613a9e00c697e8a3a5c1d38e0fbf6d00882de65c8fe9bbbd970e4fba212f181; diff:460c9a07bd781621c8349db2e9cff7e17205b173..87c0877409443ec51958a3b869eea2146525abab@sha256:5076865a048fb95b9d4de4aef6620741288280edfe4cf8f084b7ad2e856cb79a; diff:dab95e00534ba1913a6f08576dcf9791b67b36aa..7366e963b7017e3f81781d6798a3d44d1f96027c@sha256:a3ed52861966d88484445d00e8aa537739ff7071730927cb1f9ac29ffdd4e83d; diff:51d7e91abab4e1866fb208429320247f3c06a2c7..a5e348a842bdc03f0764b7fa867f73960d3aff23@sha256:0786cf3b7eca45b38df3d0b3035aba859a9c8ee40b6f20bcb1900143a3682f22;

Acceptance rollup: MET 7 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: TestInstallWritesChosenReadmePartsOnce fixes README.md as heading plus prose, LICENSE as 'MIT License', readme.parts [licence], runs Install twice, asserts the exact file (one licence part naming MIT between the heading and the prose), that the second run reports no README.md write and leaves bytes and mtime equal to the backdated first; the ahoy package passes in the export of a5e348a84.
  evidence: internal/core/ahoy/readme_apply_test.go:88 — "func TestInstallWritesChosenReadmePartsOnce(t *testing.T) {"
  evidence: internal/core/ahoy/readme_apply_test.go:97 — "want := "# Project\n\n" + readmePart("licence", "Licensed under the MIT License; see [LICENSE] (LICENSE).") + "\n\nThe owner's prose.\n""
  evidence: internal/core/ahoy/readme_apply_test.go:113 — "if mt := modTime(t, readme); !mt.Equal(old) {"
- ac-2 — MET: TestInstallLeavesHandEditedOrBrokenReadme runs the three cases (edited by hand, duplicated, unclosed), asserts README.md equals its input byte for byte, and asserts a note naming part, reason and line, including the criterion's own example string.
  evidence: internal/core/ahoy/readme_apply_test.go:123 — "func TestInstallLeavesHandEditedOrBrokenReadme(t *testing.T) {"
  evidence: internal/core/ahoy/readme_apply_test.go:131 — ""licence part edited by hand at line 4; left as it is""
  evidence: internal/core/ahoy/readme_apply_test.go:135 — ""licence part's opening mark at line 4 has no closing mark; nothing written""
- ac-3 — MET: TestReadmeDerivationMakesNoNetworkCall derives licence, release-badge and build-badge in a repo with no LICENSE and visibility private, asserts each is omitted with reasons 'no LICENSE file' and 'not declared public', and counts zero requests on a swapped http.DefaultTransport and zero dials through a forced proxy; readmeDerivedNotes carries each omission reason into the install result's notes, and the install-level test asserts 'licence omitted: no LICENSE file' there.
  evidence: internal/core/ahoy/readme_derive_test.go:815 — "func TestReadmeDerivationMakesNoNetworkCall(t *testing.T) {"
  evidence: internal/core/ahoy/readme_derive_test.go:847 — "wantOmitted(t, derivedPart(t, got, "release-badge"), "not declared public")"
  evidence: internal/core/ahoy/readme_apply.go:251 — "func readmeDerivedNotes(derived []readmeDerived) []string {"
  evidence: internal/core/ahoy/readme_apply_test.go:193 — "for _, want := range []string{"licence omitted: no LICENSE file", "links omitted: none of CHANGELOG.md"} {"
- ac-4 — MET: TestInstallCreatesReadmeWithOnlyChosenParts starts with no README.md, parts [licence, links], and asserts the created file is exactly the two marked parts, begins with '< !-- BEGIN ABCD:licence ', and records readme.created; a sub-case with nothing derivable creates no file.
  evidence: internal/core/ahoy/readme_apply_test.go:161 — "func TestInstallCreatesReadmeWithOnlyChosenParts(t *testing.T) {"
  evidence: internal/core/ahoy/readme_apply_test.go:175 — "if !strings.HasPrefix(got, "< !-- BEGIN ABCD:licence ") {"
- ac-5 — MET: TestUninstallRemovesOnlyCurrentReadmeParts asserts an owner's README with no part keeps bytes and mtime after Uninstall, and that with one current links part and one hand-edited licence part only links is removed, the receipt note reads exactly 'links removed; licence edited by hand, left' and Left names licence at line 7.
  evidence: internal/core/ahoy/readme_apply_test.go:210 — "func TestUninstallRemovesOnlyCurrentReadmeParts(t *testing.T) {"
  evidence: internal/core/ahoy/readme_apply_test.go:249 — "if receipt.Readme.Note != "links removed; licence edited by hand, left" {"
- ac-6 — MET: TestBadgeAddressRefusesForeignQueryOrHost derives both badges from an origin ending '?token=abc' and from example.com, asserts each is refused naming 'token' or 'example.com' without echoing the value, and that README.md is not written; a second table drives checkBadgeAddress through foreign query keys, hosts, schemes, credentials and fragments.
  evidence: internal/core/ahoy/readme_derive_test.go:603 — "func TestBadgeAddressRefusesForeignQueryOrHost(t *testing.T) {"
  evidence: internal/core/ahoy/readme_derive_test.go:608 — "{"https://github.com/owner/repo.git?token=abc", "token"},"
  evidence: internal/core/ahoy/readme_derive_test.go:622 — "t.Errorf("README.md was written (Lstat: %v)", err)"
- ac-7 — MET: The dated receipt records 2026-10-12 06:22:50 BST and the version ('abcd dev', local build at 15f82d0dd), runs 'ahoy install --yes --adopt' with no README flag in two fresh scratch repositories, shows README.md's sha256 and mtime identical before and after (or absent both times), and quotes the summary item saying the question was not asked and naming --readme-parts; TestInstallYesLeavesReadmeAndNamesFlag asserts the same in Go.
  evidence: .abcd/.work.local/logs/readme-parts-yes-2026-10-12.md:4 — "Date: 2026-10-12 06:22:50 BST"
  evidence: .abcd/.work.local/logs/readme-parts-yes-2026-10-12.md:12 — "`a23065984792cc89b5362e44d3e0d8ca1459af33f5ef71248af69ce5710b7405 1791782570` (untouched)"
  evidence: .abcd/.work.local/logs/readme-parts-yes-2026-10-12.md:20 — "The README question was not asked; abcd keeps no part of README.md current until it is answered."
  evidence: internal/core/ahoy/readme_apply_test.go:305 — "func TestInstallYesLeavesReadmeAndNamesFlag(t *testing.T) {"
- ac-8 — MET_WITH_CONCERNS: The receipt (2026-10-12 06:25:52 BST, harness 2.1.296, tmux 80x40) shows Q1 alone with 'Decide later' (01-q1.txt), the parts as tabs Q2-Q5 then Q6-Q7 (readme_ask.go chunks at QuestionsPerCall[1], 'one to four'), each tab with 'Now:', 'Change later:' and the exact lines it writes, and a run-2 'no' not asked in run 3 (09, 10, 11 captures). Two concerns: (1) 'a screenshot' is delivered as tmux capture-pane text files, with no image anywhere in the receipt's folder; (2) 'the managed-with-abcd badge shows as off' holds only through the shared 'Now: not set, so off' line and Q1's 'off unless you turn it on' prose, since the host's question tool rests the cursor on '1. on' in the badge tab exactly as in the licence tab whose default is on, so the badge's off default is not visibly distinct from any other unset part in the host view; it is marked only on the terminal route, which the Go tests cover.
  evidence: .abcd/.work.local/logs/readme-questions-2026-10-12.md:7 — "These are text captures of the terminal view, not image screenshots."
  evidence: /private/tmp/claude-505/-Users-[redacted-user]-ABCDevelopment-abcd/69947826-a66f-4218-b371-d8bc18fdc7c6/scratchpad/lane5-r8/captures/01-q1.txt:34 — "3. Decide later"
  evidence: /private/tmp/claude-505/-Users-[redacted-user]-ABCDevelopment-abcd/69947826-a66f-4218-b371-d8bc18fdc7c6/scratchpad/lane5-r8/captures/02-call1-tab3.txt:17 — "← ☐ Setup Q2 ☐ Setup Q3 ☐ Setup Q4 ☐ Setup Q5 ✔ Submit →"
  evidence: /private/tmp/claude-505/-Users-[redacted-user]-ABCDevelopment-abcd/69947826-a66f-4218-b371-d8bc18fdc7c6/scratchpad/lane5-r8/captures/02-call1-tab3.txt:24 — "│ Now: not set, so off"
  evidence: /private/tmp/claude-505/-Users-[redacted-user]-ABCDevelopment-abcd/69947826-a66f-4218-b371-d8bc18fdc7c6/scratchpad/lane5-r8/captures/02-call1-tab3.txt:29 — "❯ 1. on"
  evidence: /private/tmp/claude-505/-Users-[redacted-user]-ABCDevelopment-abcd/69947826-a66f-4218-b371-d8bc18fdc7c6/scratchpad/lane5-r8/captures/11-dryrun-after-no.txt:1 — "readme view present: False | readme_tool present: False | readme.offered gap: False"
  evidence: internal/core/ahoy/readme_ask.go:221 — "if len(call) == question.Default.QuestionsPerCall[1] {"
  evidence: internal/core/ahoy/readme_ask_test.go:380 — "func TestAbcdBadgeOffByDefault(t *testing.T) {"
  evidence: internal/surface/cli/ahoy_readme_ask_test.go:179 — "func TestDrawnPrompterStartsAReadmeQuestionOnItsDefault(t *testing.T) {"

Gap audit:
- honoured:
  - Setup asks whether abcd keeps README parts current, after the conventions-file question, then which parts, one per offered part
    evidence: internal/core/ahoy/readme_ask_test.go:116 — "func TestReadmeQuestionPairOrder(t *testing.T) {"
    evidence: internal/core/ahoy/readme_ask_test.go:154 — "the conventions-file question (at %d) is not asked before the README question"
  - Each chosen part lives in its own marked place and is rewritten from the project on each setup run; the owner's words are never touched
    evidence: internal/core/ahoy/readme_apply_test.go:88 — "func TestInstallWritesChosenReadmePartsOnce(t *testing.T) {"
  - A hand-edited part is reported and left; a duplicated or unclosed mark refuses the write naming the line
    evidence: internal/core/ahoy/readme_apply_test.go:123 — "func TestInstallLeavesHandEditedOrBrokenReadme(t *testing.T) {"
  - A project with no README gets one holding only the chosen parts, and no title is ever written
    evidence: internal/core/ahoy/readme_apply_test.go:161 — "func TestInstallCreatesReadmeWithOnlyChosenParts(t *testing.T) {"
  - Release and build badges are offered only to a project declared public; the managed-with-abcd badge is offered everywhere and off by default
    evidence: internal/core/ahoy/readme_derive_test.go:520 — "func TestBadgePartsNeedPublicAndForge(t *testing.T) {"
    evidence: internal/core/ahoy/readme_ask.go:77 — ""release-badge": true, "build-badge": true, "abcd-badge": false,"
  - Links name only files the project has (changelog, contributing guide, security policy, and one more), and the table of contents is off unless wanted
    evidence: internal/core/ahoy/readme_derive_test.go:128 — "func TestLinksPartListsOnlyFilesThatExist(t *testing.T) {"
    evidence: internal/core/ahoy/readme_ask.go:78 — ""licence": true, "links": true, "contents": false,"
  - Uninstall removes only the parts abcd can prove it wrote
    evidence: internal/core/ahoy/readme_apply_test.go:210 — "func TestUninstallRemovesOnlyCurrentReadmeParts(t *testing.T) {"
  - A no is stored and not asked again; deciding later stores nothing; a non-interactive run declines and names the flag
    evidence: internal/core/ahoy/readme_ask_test.go:208 — "func TestReadmeNoIsStored(t *testing.T) {"
    evidence: internal/core/ahoy/readme_ask_test.go:336 — "func TestReadmeLaterStoresNothing(t *testing.T) {"
    evidence: internal/surface/cli/ahoy_readme_test.go:16 — "func TestAhoyInstallReadmePartsFlag(t *testing.T) {"
  - The how-to and the brief describe the parts, the question pair, the flag, --yes and uninstall
    evidence: docs/how-to/install.md:494 — "## The README parts abcd keeps current"
    evidence: docs/how-to/install.md:596 — "`--yes` never answers the README question."
- diverged:
  - R8's 'a screenshot' is delivered as tmux capture-pane text files; no image screenshot exists in the receipt or its capture folder
    evidence: .abcd/.work.local/logs/readme-questions-2026-10-12.md:7 — "These are text captures of the terminal view, not image screenshots."
  - R8's 'the managed-with-abcd badge shows as off' is shown in the host's question tool only by the 'Now: not set, so off' line every unset part carries and by Q1's prose; the cursor rests on 'on' in the badge tab as in the licence tab, so the off default is marked only on the terminal route
    evidence: /private/tmp/claude-505/-Users-[redacted-user]-ABCDevelopment-abcd/69947826-a66f-4218-b371-d8bc18fdc7c6/scratchpad/lane5-r8/captures/02-call1-tab3.txt:29 — "❯ 1. on"
    evidence: .abcd/.work.local/logs/readme-questions-2026-10-12.md:243 — "│ Now: not set, so off"
    evidence: internal/surface/cli/ahoy_readme_ask_test.go:179 — "func TestDrawnPrompterStartsAReadmeQuestionOnItsDefault(t *testing.T) {"
- missing: (none)

Scope-condition dispositions:
- cond-2610031704331158 — survived: The only README the parts are planned into, written to, created or removed from is the constant readmeFileName joined to the repository root; the contents part reads README.md in the root alone.
  evidence: internal/core/ahoy/readme_apply.go:45 — "const readmeFileName = "README.md""
  evidence: internal/core/ahoy/readme_derive.go:555 — "data, err := readInRoot(r.root, "README.md")"
- cond-2610031704332577 — survived: readmeSettings.Visibility comes from repo.visibility in .abcd/config.json and is the only source of 'public'; a private or unset declaration omits both forge badges with the reason naming the declaration and that no visibility is read from the forge, asserted by TestBadgePartsNeedPublicAndForge and shown live in the R8 receipt's public scratch project.
  evidence: internal/core/ahoy/readme_derive.go:41 — "(`repo.visibility`), the only source of "public" (scope condition 2)."
  evidence: internal/core/ahoy/readme_derive.go:242 — "return forgeRemote{}, "not declared public (repo.visibility in .abcd/config.json is " + vis + "; abcd reads no visibility from the forge)""
  evidence: internal/core/ahoy/readme_derive_test.go:531 — "t.Run("not declared public", func(t *testing.T) {"
- cond-2610031704332960 — narrowed: The licence is named from an SPDX identifier line or a forge-standard first line of a fixed table, otherwise linked unnamed, and no LICENSE at the root yields no part, as the condition says; but the identifier line is honoured only within the file's first twenty lines, and one past that leaves the licence linked without a name.
  narrowing: Holds for an SPDX-License-Identifier line within the first twenty lines of the LICENSE file (LICENSE, LICENSE.txt or LICENSE.md at the root) and for the forge-standard first lines in the fixed table; an identifier line later in the file names nothing.
  evidence: internal/core/ahoy/readme_derive_test.go:74 — "func TestLicencePartNamesOnlyFromIdentifierOrFirstLine(t *testing.T) {"
  evidence: internal/core/ahoy/readme_derive_test.go:105 — "{"identifier past the twentieth line", map[string]string{"LICENSE": padding + "SPDX-License-Identifier: MIT\n"}, unnamed},"
  evidence: internal/core/ahoy/readme_derive_test.go:111 — "{"no LICENSE at the root", map[string]string{"docs/LICENSE": "MIT License\n", "COPYING": "MIT License\n"}, ""},"
- cond-2610031704338772 — survived: The parts are written only inside Install's apply pass and removed only inside Uninstall; no code outside internal/core/ahoy (the CLI flag plumbing aside) touches them, a decide-later or a --yes run stores no answer, and the summary leaves the commit to the person ('Commit README.md.').
  evidence: internal/core/ahoy/readme_apply.go:364 — "a.note(writeReadmeParts, path)"
  evidence: internal/core/ahoy/readme_apply.go:466 — "res, err := removeReadmeParts(path)"
  evidence: internal/core/ahoy/install_summary.go:140 — "Action: "Commit README.md. A part you edit by hand is left as it is;"
  evidence: internal/core/ahoy/readme_apply_test.go:339 — "t.Errorf("--yes stored a README answer")"
- cond-2610031704334415 — survived: Derivation of every part in both a private and a public project hands zero requests to http.DefaultTransport and opens zero connections through a forced proxy, and readme_derive.go and readme_check.go import no network package; the badges are shields.io and forge badge.svg addresses the forge draws when the page is read.
  evidence: internal/core/ahoy/readme_derive_test.go:861 — "t.Errorf("derivation handed %d request(s) to http.DefaultTransport", n)"
  evidence: internal/core/ahoy/readme_derive.go:280 — "return badgeMarkdown("Managed with abcd", img, abcdPublicRepo), []readmeAddr{{readmeAddrImage, img}, {readmeAddrLink, abcdPublicRepo}}, """
<!-- abcd-review-end receipt=rcp-5b7cb4b81566 -->

## Grounds

- pursued: owners get help on their README only where they ask for it (the product thinker, 2026-10-03).
