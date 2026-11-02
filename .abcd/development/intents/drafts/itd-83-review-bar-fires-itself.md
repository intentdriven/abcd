---
id: itd-83
slug: review-bar-fires-itself
spec_id: null
kind: null
suggested_kind: standalone
reclassification_history: []
related_adrs: []
prd_path: null
severity: major
builds_on: [itd-3, itd-81]
---

# The Review Bar Fires By Itself, In Every Repo abcd Manages

## Press Release

> **abcd's reviewer agents now invoke themselves at the moment they are needed.** A
> `REVIEW` domain in the bundled rules loader recalls on the language of shipping —
> *review*, *diff*, *PR*, *present*, *ship*, *release* — and injects the standing
> rule just-in-time: a non-trivial diff gets `abcd:ruthless-reviewer` before it is
> presented; a change touching a trust boundary gets `abcd:security-reviewer` before
> it lands; a release gets `abcd:docs-currency-reviewer`. Nothing to remember,
> nothing to configure, no text copied into your repo. Install abcd and the review
> bar is simply on.
>
> "I kept forgetting to run the security reviewer on exactly the changes that needed
> it most — the ones I was in a hurry to ship," said Carol, a solo maintainer. "Now
> it just happens. The one time it blocked me, it was right."

## Why This Matters

[itd-81](../disciplines/itd-81-judge-calibration.md) shipped four reviewer agents
into the plugin, and they are *reachable* — `abcd:ruthless-reviewer` and friends
resolve in every repo with abcd enabled. But nothing **calls** them. They fire only
when the host model happens to choose to delegate, or when a human remembers to ask.
Under abcd's own **wired-or-it-isn't-done** rule, that is not done: an agent nothing
invokes is scaffolding with a good description.

The rule that makes them fire currently lives in the maintainer's **personal
machine-level instructions** — a file no abcd user has, in a directory no abcd user
has. Every other repo abcd manages gets the agents and none of the discipline. The
capability is shipped; the reason to use it is not.

abcd already owns the right mechanism. The modular rules loader ([itd-3](../shipped/itd-3-modular-rules-loader.md))
ships eight bundled domains with keyword recall, injects matched rules just-in-time
on `UserPromptSubmit` via a wired hook, and is per-repo overridable through
`.abcd/rules.json`. `COMMITTING` already carries standing rules of exactly this shape
("Substantive work goes on a branch + PR", "never force-push"). A review bar is the
same kind of rule, and it belongs in the same place — not copied into every repo's
`AGENTS.md`, which would be a fourth copy of text that has a canonical home.

## What's In Scope

- A **`REVIEW` domain** in the bundled `rules.json`, `state: active`, recalling on
  the vocabulary of shipping (`review`, `diff`, `pull request`, `present`, `ship`,
  `release`, `security`, `merge`) with the standing rules that name the agents:
  ruthless before presenting a non-trivial diff; security on any trust boundary
  (auth, secrets, input parsing, network, subprocess); docs-currency before a
  release tag. Whether a security BLOCK is a hard stop or advisory is an open
  question below — it turns on whether the agent has been measured.
- Rules **point at** the agents and the canonical discipline
  ([itd-81](../disciplines/itd-81-judge-calibration.md)); they never restate its
  content. This follows the `OPINIONS` precedent, whose rules point at
  `.abcd/development/principles/` rather than copying them.
- A **host-agnostic fallback**: a one-line pointer in the marked conventions section
  of `AGENTS.md` that `prepare-this-repo` already owns, so a harness that does not
  run the hook still learns the agents exist. A **pointer, not a restatement** — if
  the line grows into a copy of the rules, the design has failed.
- Per-repo override: a repo that wants a different bar (or none) turns the domain off
  or replaces its rules through `.abcd/rules.json`, using the loader's existing
  per-field override and sticky kill switch.

## What's Out of Scope

- **Enforcement.** The domain *injects a rule*; it does not gate a commit. A hook
  that hard-blocks a commit until a reviewer has run is a separate, heavier
  capability, and it would need [itd-81](../disciplines/itd-81-judge-calibration.md)'s
  corpus scores first — enforcing an unmeasured judge is exactly the failure that
  discipline exists to prevent.
- **Auto-fixing what a reviewer finds.** Findings go to the human.
- **Calibrating the reviewers.** That is itd-81's corpus work, and it is a hard
  prerequisite for trusting anything this intent causes to fire.
- **Changing the agents' prompts.** They ship as-is at `0.1.0`.

## Acceptance Criteria

> _BDD format, per the [itd-1 discipline](../disciplines/itd-1-acceptance-gates.md)._

- **Given** a repo with the abcd plugin enabled, **when** `abcd rules REVIEW` runs,
  **then** it renders the review domain's rules, each naming the agent it invokes.
- **Given** a session in an abcd-managed repo, **when** the operator's prompt matches
  the review domain's recall vocabulary, **then** the prompt-router hook injects the
  review rules exactly once per session signature (the loader's existing dedup).
- **Given** a repo whose `.abcd/rules.json` disables or overrides the `REVIEW`
  domain, **when** the loader runs, **then** the repo's choice wins and the bundled
  default does not reappear.
- **Given** an abcd-managed repo prepared by `prepare-this-repo`, **when** its
  `AGENTS.md` conventions section is read, **then** it points at the reviewer agents
  in one line and does **not** restate the rules the loader injects.
- **Given** a harness that does not run the prompt-router hook, **when** an agent
  reads `AGENTS.md`, **then** it can still discover that the reviewer agents exist
  and what each is for.
- **Given** this intent ships, **when** the wired-or-it-isn't-done check runs, **then**
  each of `abcd:ruthless-reviewer`, `abcd:security-reviewer`, and
  `abcd:docs-currency-reviewer` is demonstrably invoked from a production path, not
  merely present in `agents/`.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in
the [forge, CI and merge SOTA note](../../research/notes/2026-10-23-forge-ci-merge-sota.md).
The host-agnostic fallback was surveyed in the
[AGENTS.md consumers note](../../research/notes/2026-09-30-claude-md-consumers-and-agents-md-sota.md)
and its [recheck](../../research/notes/2026-10-03-agents-md-native-reading-recheck-sota.md)
and is not redone.

- **Measured uptake of automated review comments is low, and precision is the
  stated reason**: CodeRabbit comments across 239 repositories were 36.4%
  accepted, 7.3% discussed, 56.3% rejected, rejections attributed to
  "invalid suggestions that were false positives, redundant, or out of scope"
  (Lin et al., arXiv 2607.03316); across 13 review agents, agent-only pull
  requests merged at 45.2% against 68.4% for human-only review and 12 of 13
  agents averaged a signal ratio below 60% (Chowdhury et al., MSR 2026);
  over 278,790 review conversations, agent suggestions were adopted at a
  significantly lower rate than human ones and over half of the unadopted
  were incorrect or were fixed another way (Zhong et al., arXiv 2603.15911).
- **What gets acted on**: comments that are "concise, include code snippets,
  and are manually triggered" (Sun et al., arXiv 2508.18771, 16 review
  Actions, 22,000 comments). A rule that fires on the words of shipping is
  nearer "manually triggered" than a bot on every push.
- **The calibrated-precision precedent**: Google shipped ML-suggested edits
  at a 50% target precision, with 40–50% of previewed suggestions applied.
- **Review capacity is what the bar competes for**: under an enterprise "2×"
  mandate, "per-reviewer load roughly doubled and automated review overtook
  human review", with merge and revert rates unchanged (He et al., arXiv
  2607.01904).
- **The forge's own self-firing reviewer is advisory by default**: Copilot
  code review is requestable automatically by repository, organisation or
  ruleset; "By default, its reviews do not count toward required approvals";
  a preview lets its approval satisfy a required-approval rule. A live
  `copilot_code_review` ruleset rule was read anonymously on a public
  repository on 2026-10-23.

**Path 2**: a native review rule fired inside the host through the rules
loader, with an opt-in forge-side reviewer adapter as the seam kept open for
later; the reason the field's forge-bot shape is not adopted first is abcd's
host-delegated boundary and the basics-built-in principle (a deferred seam is
still Path 2 under sota-per-intent, so no complexity stop is invoked). The
evidence supports the record's leaning on the open question it carries, ship
the security rule as advisory until a measured true-negative rate exists
(low measured precision; the forge's own self-firing reviewer is advisory by
default); the call stays the maintainer's. The independent fit-challenge runs
at plan time.

## Open Questions

- **One `REVIEW` domain, or rules folded into the existing `COMMITTING` domain?**
  `COMMITTING` already recalls on `pr`/`branch`/`merge` and already carries
  branch-and-PR rules, so a reviewer rule is arguably at home there — but review also
  fires on `release` and `security`, which `COMMITTING` does not recall. Leaning
  toward a separate domain; confirm at spec time by looking at real recall overlap.
- **Does the security rule's BLOCK belong in a rule at all,** given the agent is
  unmeasured (`0.1.0`)? A rule that says "a BLOCK stops the change" grants authority
  to a judge whose false-positive rate is unknown. Options: ship the rule as advisory
  until itd-81 produces a TNR figure, or ship it as a stop and accept the noise.
  **Recommend advisory until measured** — but this is a maintainer call.
- **Host-agnosticism.** The loader's injection path is a Claude Code hook. Naming the
  fallback "`AGENTS.md`" assumes other harnesses read that file. Is that assumption
  good enough, or does the fallback need its own surface?
- (SOTA 2026-10-23) Whether the rule asks the reviewer for one concise,
  snippet-bearing finding set rather than a running commentary, since that
  is the shape the measurements say gets acted on.
- (SOTA 2026-10-23) Where a managed repository already has the forge's
  self-firing reviewer (a `copilot_code_review` ruleset rule), whether the
  review bar reports it as part of the bar or ignores it.

## Dependencies

- **Hard prerequisite:** [itd-3](../shipped/itd-3-modular-rules-loader.md) — the rules
  loader, its domains, and the prompt-router hook. Phases 1 + 3 have shipped; this
  intent adds a domain to what exists and builds no new machinery.
- **Hard prerequisite:** [itd-81](../disciplines/itd-81-judge-calibration.md) — the
  agents this intent wires, and the discipline that says they must be measured before
  they are trusted. Wiring an unmeasured judge into a hard gate is out of scope for
  exactly that reason.
- **Coordinated with:** `prepare-this-repo` — it owns the marked `AGENTS.md`
  conventions section the fallback pointer lands in.

## Audit Notes

_Empty. Populated by `intent-fidelity-reviewer` when this intent ships._

## References

- [`itd-81-judge-calibration.md`](../disciplines/itd-81-judge-calibration.md) — the
  discipline that shipped the agents and governs when they may be trusted.
- [`itd-3-modular-rules-loader.md`](../shipped/itd-3-modular-rules-loader.md) — the
  mechanism this intent extends.
- [`itd-5-prompt-quality-additions.md`](../disciplines/itd-5-prompt-quality-additions.md)
  — the prompt-quality floor every agent named here inherits.
