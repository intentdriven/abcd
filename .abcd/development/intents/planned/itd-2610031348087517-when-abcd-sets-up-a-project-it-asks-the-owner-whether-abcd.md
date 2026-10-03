---
id: itd-2610031348087517
slug: when-abcd-sets-up-a-project-it-asks-the-owner-whether-abcd
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

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: owners get help on their README only where they ask for it (the product thinker, 2026-10-03).
