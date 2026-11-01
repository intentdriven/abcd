---
id: itd-91
slug: ai-attribution-preference
spec_id: null
kind: null
suggested_kind: standalone
reclassification_history: []
related_adrs: []
prd_path: null
severity: minor
---

# A Project Declares Once How It Wants AI Acknowledged — And Every Commit, PR, And Record Follows

## Press Release

> **abcd asks how you want AI credited, then makes every artefact obey.** When you set up a project, abcd now asks a single question — how should AI assistance be acknowledged? — and offers the honest options: not at all, a kernel-style `Assisted-by:` trailer, a `Co-Authored-By:` line, a `Generated-by` note, or your own wording. Your answer is recorded once, in the repo, and from then on every commit trailer, pull-request body, and generated record uses exactly that convention. No more a policy that lives only in a contributor's memory and drifts PR by PR; no more a tool stamping its own name where a project never asked it to.

> "We'd written the rule down in CONTRIBUTING, and it still drifted — half our PRs said one thing, half said another, and one tool kept adding its own footer we never agreed to," said Carol, a tech lead standardising a team's repos. "Now it's a setup question. I pick kernel-style once, and everything downstream is consistent. When a teammate preferred no attribution at all on their fork, they picked that, and abcd just honoured it. The convention stopped being folklore."

## Why This Matters

abcd already has an opinion about AI attribution — `Assisted-by: Claude:<model>`, the kernel format, as disclosure not authorship, never `Co-Authored-By:` (which asserts an authorship the tool does not hold and inflates the contributor graph). That opinion is written in `CONTRIBUTING.md` and this repo's own `CLAUDE.md`, and it is right *for this project*. But it lives as prose a human has to remember and apply by hand, and this session proved the cost: PR bodies drifted to a tool's default "Generated with" footer, and a sweep of 78 pull requests was needed to reconcile them — some carrying a model, some none, one carrying the wrong model entirely.

Two problems compound. First, **the convention is unenforced**: nothing at commit or PR time checks that the acknowledgement matches what the project decided, so it degrades with every contributor and every tool that has its own idea. Second, **the convention is not portable**: a project adopting abcd inherits *abcd's* preference by reading its docs, but has no first-class way to declare its *own* — a team that wants `Co-Authored-By:`, or wants no attribution at all, or wants a house style, has to fork the prose.

An onboarding step (`/abcd:init` or `prepare-this-repo`) is the right moment to resolve both. Setup is already where a project's conventions are established; adding "how do you want AI acknowledged?" makes the choice explicit, records it as durable per-repo config, and lets the rest of abcd — commit-trailer guidance, PR-body composition, record generation, and any attribution lint — read that one source of truth instead of assuming a default. The project decides once; the tooling obeys everywhere.

## What's In Scope

- **A setup question** in the onboarding surface (`/abcd:init` / `prepare-this-repo`) offering a selection of acknowledgement conventions: **none**, **kernel-style `Assisted-by:`** (the current abcd default), **`Co-Authored-By:`**, **`Generated-by`**, and a **custom** free-form template.
- **Durable per-repo config** capturing the choice (and, for the trailer styles, whether a model identifier is included), stored in the repo so it travels with it and is not a contributor's memory.
- **A single read seam** the rest of abcd consults when it composes a commit trailer, a PR body, or a generated record — so the recorded preference is actually applied, not merely stored.
- **An honest default and honest framing.** The default remains the disclosure-not-authorship position (kernel `Assisted-by:`), and the `Co-Authored-By:` option carries the caveat that it asserts authorship — the selector informs the choice, it does not hide the trade-off.

## What's Out of Scope

- **Rewriting history.** This governs artefacts produced *after* the choice is made; reconciling a repo's existing commits/PRs to a new convention is a separate, deliberate migration (this session did the PR-body form of it by hand), not something setup performs.
- **A commit/PR-time enforcement lint.** Checking that a produced trailer matches the recorded preference is a natural follow-up (an attribution rule in the record-lint family), but this intent is the *declaration and application* seam; the *gate* can build on it later.
- **Deciding abcd's own house policy.** abcd keeps its `Assisted-by:` stance for its own repo (`CONTRIBUTING.md`); this intent gives *adopting* projects a way to choose theirs, it does not relitigate abcd's.
- **Attribution beyond AI.** Human `Signed-off-by:` (DCO) and the `ACKNOWLEDGEMENTS.md` credit ledger are their own conventions; this intent is scoped to how *AI assistance* is acknowledged.

## Acceptance Criteria

> _Given-When-Then per the itd-1 discipline._

- **Given** a project running the onboarding surface, **when** the attribution question is presented, **then** it offers at least {none, kernel `Assisted-by:`, `Co-Authored-By:`, `Generated-by`, custom} and records the chosen convention as durable per-repo config.
- **Given** a recorded preference of "none", **when** abcd composes a commit trailer or PR body, **then** it adds no AI-attribution line at all.
- **Given** a recorded preference of kernel `Assisted-by:` with a model identifier, **when** abcd composes an artefact, **then** the artefact carries exactly `Assisted-by: Claude:<model>` and no `Co-Authored-By:` or "Generated-by" line.
- **Given** a recorded `Co-Authored-By:` preference, **when** it is selected, **then** the selector surfaces the caveat that it asserts an authorship the tool does not hold — the trade-off is disclosed, not buried.
- **Given** no recorded preference (a repo predating this feature), **when** abcd composes an artefact, **then** it falls back to the documented default (kernel `Assisted-by:`) rather than a tool's own footer.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [sources-provenance SOTA note](../../research/notes/2026-10-23-sources-provenance-sota.md). The repository's `ATTRIBUTION` rules settle abcd's own policy; this surveys only the configurable-preference gap.

- **The field has forked three ways and all three are current.** Disclosure trailer: the Linux kernel's document at 7.2 reads `Assisted-by: AGENT_NAME:MODEL_VERSION [TOOL1] [TOOL2]` (example `Assisted-by: Claude:claude-3-opus coccinelle sparse`); the current document (7.3) reads `Assisted-by: LLM [TOOL1] [TOOL2]` and drops the model; Rocky Linux makes disclosure MUST for unchanged tool output and SHOULD otherwise, with `Assisted-by:` as the recommended method; LLVM asks contributors to label substantial tool-generated content and gives `Assisted-by:` as its example trailer, with no MUST/SHOULD split. Co-authorship: Claude Code's default is `Co-Authored-By: <model> <noreply@anthropic.com>`, VS Code's `git.addAICoAuthor` went from `off` (1.110) to on by default for chat and agent workflows (1.118), and GitHub's cloud agent makes Copilot the author with the human as co-author. Prohibition: Gentoo (council vote 14 April 2024), QEMU (DCO grounds, exceptions by list approval), NetBSD (presumed tainted, core approval), and git's SubmittingPatches (no trailer; rejects what "looks AI generated").
- **Hosts already expose the preference as configuration.** Claude Code's `attribution` key takes `commit` (string, empty hides), `pr` (string, empty hides) and `sessionUrl` (Boolean; a `Claude-Session` trailer), or `false`, settable in any file including the committed project settings file; `includeCoAuthoredBy` is deprecated since v2.0.62. User and local settings sit above the committed file, so a committed preference is a default, not an enforcement; `sessionUrl` applies only to cloud and Remote Control sessions; and the host tells the model that a CLAUDE.md or memory rule about attribution takes precedence over these commit and PR lines unless they are managed settings, so abcd's own instruction-file rule is itself one of the compile targets (Claude Code settings reference).
- **git has the application machinery**: `trailer.<alias>.key`, `ifExists` (`replace`, `doNothing`, default `addIfDifferentNeighbor`), `ifMissing`, `trailer.<alias>.cmd`, and the manual's own `commit-msg` hook running `git interpret-trailers` over the message file, which is the one place a human's own `git commit` is reached.
- **"None" is two stances.** The draft's `none` covers both "do not disclose" and "AI use is prohibited"; Gentoo, QEMU and NetBSD show the second is a policy of its own.
- **Where the research contradicts the record**: the house rule names "kernel format" with a `Vendor:model` value, and the kernel's current document has moved to `LLM` plus tools without the model. The ruling stands; the selector should name the two value forms separately rather than call either "kernel-style".
- **Path 1 for the surfaces, Path 2 for the record**: one small structure in `.abcd/config.json` (`stance` of `none | disclose | coauthor | custom | prohibit`, `trailer_key`, `value_form`, `pr_text`; `Generated-by` is a `disclose` preset with `trailer_key: Generated-by`, so the criterion's five offered options stay and the stance set is the stored form, not the menu), compiled outward in the same act to the host's `attribution.commit`, `attribution.pr` and `attribution.sessionUrl: false`, to `trailer.*` git config plus a `commit-msg` hook that normalises whatever the host or human wrote into the declared form, and to the PR-body composer. The seam is the compiler: a new host adds one target. No Go dependency. The independent fit-challenge runs at plan time.

## Open Questions

- (SOTA 2026-10-23) A project that picks `coauthor` chooses to fail abcd's own attribution gate; does the selector disable that gate for the project, or state the conflict and keep the gate abcd-only?
- (SOTA 2026-10-23) Writing `attribution` into the host's committed settings file is writing into a directory abcd does not own; is it a marked block with consent, as the `AGENTS.md` conventions section is, or a printed instruction?
- (SOTA 2026-10-23) Does `prohibit` compile to anything beyond the hook refusing a disclosure trailer, such as a refusal of a known tool footer in a PR body?
- **Where does the preference live?** A dedicated key in an existing per-repo config (e.g. `.abcd/config/…`) versus a new file; and whether it is one setting or a small structure (style + include-model + custom-template).
- **How is it applied to commits abcd does not author?** abcd can compose its own PR bodies and records, but a human's `git commit` is outside its control — is the mechanism a documented trailer the human/agent is told to use, a prepare-commit-msg hook, or guidance only?
- **One convention or per-artefact?** A project might want `Assisted-by:` on commits but nothing in PR bodies; whether the selector is a single global choice or per-artefact granularity is a design call (start single, likely).
- **Does the custom template need variables?** A free-form template implies placeholders (`{model}`, `{date}`); how much templating to support before it becomes its own small language.
- **Relationship to a future enforcement lint.** If an attribution record-lint rule follows, this config is its source of truth — the schema should be chosen with that consumer in mind even though the lint is out of scope here.
