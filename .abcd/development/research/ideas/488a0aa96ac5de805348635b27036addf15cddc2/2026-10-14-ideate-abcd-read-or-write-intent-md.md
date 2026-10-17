# Ideate verdict — abcd-read-or-write-intent-md

**Verdict: reframed.** Recorded on 2026-10-14 by abcd's idea-admission protocol —
primary-source research, a grill against the existing record, and an
independent adversarial review. This record exists so the idea is not
re-litigated: it stands whether the idea lived or died.

## The idea

should abcd read or write Anthropic's intent.md?

## Leg 1 — Primary-source research

Every load-bearing claim checked against its primary source, never a
secondary citation.

| Claim | Primary source | Finding |
|---|---|---|
| Anthropic published intent.md as the first committed artifact of its AI-native SDLC playbook (intent.md -> spec.md -> plan.md -> tests -> PR -> incident -> new intent.md), on 21 August 2026. | https://claude.com/blog/the-ai-native-sdlc-playbook | verified |
| intent.md has a defined schema abcd could target. | https://claude.com/blog/the-ai-native-sdlc-playbook | falsified |
| The post's sections (Problem, Proposed outcome, Affected users and systems, Constraints, Open questions, plus an Author/Status line) are an example of an organization-defined template, optionally packaged as a skill. | https://claude.com/blog/the-ai-native-sdlc-playbook | verified |
| Claude Code reads or discovers intent.md as a built-in feature. | https://code.claude.com/docs/llms.txt | falsified |
| The suggested home is an intent/ folder in the product repo; acceptance is the merge of the artifact or a closed review, the commit chain is the audit trail, and intent.md carries no record id. | https://claude.com/blog/the-ai-native-sdlc-playbook | verified |
| intent.md has no acceptance-criteria section. | https://claude.com/blog/the-ai-native-sdlc-playbook | verified |
| Practitioners are adopting intent.md files in real repositories at a scale that creates demand for interop. | https://claude.com/blog/the-ai-native-sdlc-playbook | unverifiable |

## Leg 2 — Record grill

Does the brief, an intent, an ADR, or a principle already cover,
contradict, or supersede this idea? Every hit is cited by record id, and
every id resolved in this repository when the verdict was recorded.

| Record | Relation | Note |
|---|---|---|
| itd-23 | covered | The draft Spec Kit interop intent is the foreign-format adapter family (embark from-spec-kit, disembark to-spec-kit, stub Linear/Jira/Notion adapters); an intent.md import is one more instance. Its out-of-scope keeps abcd's source of truth in .abcd/development/intents/, and one criterion refuses an unsupported external format such as raw Markdown, which intent.md is. The brief lists from-spec-kit and to-spec-kit as not built (04-surfaces/03-embark.md, 02-disembark.md, 06-delivery/03-out-of-scope.md); the research note spec-kit-vs-flow-next.md calls Spec Kit a worthy import/export peer and rejects adopting it wholesale. |
| adr-1 | contradicted | An abcd intent is press-release-shaped with a persona quote, scope and Given-When-Then acceptance criteria; adopting intent.md as abcd's format contradicts it. |
| itd-30 | contradicted | The only record weighing a second intent format keeps it abcd-native and rejects auto-conversion: auto-conversion would lose the texture. |
| iss-92 | covered | Onboarding should classify a repository's non-standard files (canon, equivalent, foreign) and propose where each belongs; an intent.md already in a repository falls under it. |

## Leg 3 — Adversarial review

Conducted fresh-context and off-policy by an evaluator that did not carry
out the research and received the idea as an artefact of unknown
authorship — the evaluator-outside-the-loop principle applied to ideas.

- **fatal** — Adopt: there is no format to adopt; the template is organization-defined and the post gives one example.
- **fatal** — Adopt: intent.md has no acceptance criteria or persona quote, which abcd's build loop, auditor and adr-1 depend on.
- **fatal** — Export as a standing feature starts a second artifact chain (intent.md -> spec.md -> plan.md) beside abcd's own specs and plans, contradicting itd-23's single source of truth.
- **partial** — Import is lossy: the press release and acceptance criteria must be invented or left empty.
- **partial** — With no schema, a Go importer either guesses headings (a write verb guessing is a defect) or fits only the blog's example.
- **partial** — An existing record already covers it: itd-23 is the same shape, and its raw-Markdown refusal criterion contradicts an intent.md import as proposed.
- **partial** — Nobody needs it: abcd intents are already Markdown any agent reads, a host converts either way in one conversation, and adoption is unverifiable.
- **partial** — Moving target: a seven-week-old playbook, not a specification, with an organization-defined template.
- **partial** — Vendor lock-in: a feature named after one vendor's playbook breaks host-agnosticism.
- **survived** — Export as a one-way render is cheap: the five sections map onto Why This Matters, Press Release, persona, scope and Open Questions.
- **survived** — Reframe that survives both reviewers: no adoption and no standing export; fold intent.md into itd-23 and iss-92 as one case of a host-agnostic 'foreign Markdown intent' import, where the host converts each file into an abcd draft the person confirms, missing acceptance criteria become open questions, nothing names a vendor, and nothing is built until a real intent.md file or a requester appears.

## Rejected alternatives

- **Adopt intent.md as abcd's own intent format.** — There is no format to adopt, and it drops the press release and acceptance criteria abcd requires (adr-1, itd-30).
- **A standing export verb that writes intent.md.** — It starts a parallel artifact chain beside abcd's specs and plans; a one-off handoff is a host conversation, not a verb.
- **A Go parser for the blog's five headings.** — It guesses on a write or fits one organization's template only.
- **A dedicated verb named for intent.md or Anthropic.** — Naming one vendor breaks host-agnosticism, and itd-23's adapter family already holds the shape.
- **Two-way sync with an intent/ folder.** — Lossy both ways, itd-30 rejects auto-conversion, and acceptance means a merge on one side and a folder move on the other.
- **Adding spec.md and plan.md support at the same time.** — Speculates on a chain no schema defines.

## What follows

The idea as posed does not survive, but the reframing recorded above
does. Any graduation to a draft intent carries the reframing, not the
original wording.
