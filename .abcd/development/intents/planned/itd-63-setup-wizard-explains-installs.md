---
id: itd-63
slug: setup-wizard-explains-installs
spec_id: null
kind: bundle-member
bundle: spc-83-operator-surfaces
suggested_kind: standalone
reclassification_history: []
related_adrs: [adr-22]
prd_path: null
prd_grandfathered: true
grandfathered: true
grandfathered_at_phase: phase-5-roundtrip
glossary_terms_used:
  - core/oracle
  - core/intent
  - distribution/end-user
  - distribution/release
builds_on: [itd-62]
severity: major
---

# The Amateur Coder Is Told What Is Being Installed And Why, Not Just Asked To Run A Command

## Press Release

> **abcd gains a setup wizard: when a capability offers an optional adapter the amateur has not installed — or genuinely needs an external tool (a security scanner, a runtime, a CLI) — abcd explains in plain language WHAT the adapter or tool is, WHY the capability would use it, what its native default already covers, and what the tool does — then guides any install — instead of dumping a command the product thinker cannot evaluate.** Most bundled dependencies are optional adapters with native defaults ([adr-22](../../decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md)), so the wizard's first job is to explain that a capability already works on its native default and an adapter is an upgrade, not a prerequisite. Where a capability does have a hard external prerequisite — abcd's safety gate always blocks on a missing scanner — the thesis keeps human judgment the constraint, but a human cannot judge "install a scanner" if they do not know what it is or why their safety gate needs it. The wizard turns an opaque prerequisite into an informed choice: it names the tool, states the capability that requires it (e.g. "the security gate needs this to scan your app for vulnerabilities"), links what it is, shows the exact install step, and confirms before proceeding.

> "I'm fine installing things — I'm not fine installing things I don't understand," said a product thinker setting up abcd's safety gate. "Tell me this is a security scanner, that my safety check can't run without it, and what it'll do. Then I'll say yes. Don't just throw a command at me and assume I know."

## Why This Matters

abcd's safety gate (itd-62/spc-76) ALWAYS blocks on a missing scanner rather than degrading to advisory — the right call for the guarantee, but it puts an install prerequisite in front of an amateur who may not recognise the tool. The thesis says keep the human's judgment the constraint; an install prompt the human cannot evaluate is judgment removed, not preserved. A setup wizard restores it: by explaining what and why, it lets the product thinker make an INFORMED decision rather than a blind one. This is a general need. Most abcd capabilities run on a native default and expose an optional adapter (adr-22) — the wizard explains that the native default already works and names what the adapter would add, so an optional install is never mistaken for a requirement. The few capabilities with a genuine external prerequisite (a security scanner, a runtime, a CLI, a model) get the same explain-then-install surface; itd-62 is the first caller.

## What's In Scope

- A reusable setup-wizard surface that, given a missing optional adapter or a genuine prerequisite, presents: the tool/adapter name, the capability that uses it (for an adapter, the native default that already covers it; for a prerequisite, what fails without it), a plain-language description of what the tool does, the exact install step, and a confirmation.
- Integration as the install-guidance path for itd-62/spc-76's "always block on missing scanner" (its first consumer).
- Honesty about what the install does to the machine (and what it does NOT do), so the human consents knowingly.
- Local-first, no Claude-Code dependency for the explain-and-guide mechanics.

## What's Out of Scope

- Silently auto-installing dependencies without informed confirmation (the whole point is the human decides).
- Bypassing or weakening a gate's fail-closed guarantee — declining an install still blocks; the wizard informs, it does not downgrade.
- Re-implementing package managers — it guides the human (or runs a confirmed, explained step), it is not a new installer.

## Scope Conditions

None stated.

## Acceptance Criteria

> _Given-When-Then per the itd-1 discipline._

- **Given** a capability needs a dependency the machine lacks (an optional adapter or a genuine prerequisite), **when** the wizard runs, **then** it states the tool name, the requiring capability, the native default already covering it (for an adapter) or what fails without it (for a prerequisite), a plain-language description, and the exact install step before any install.
- **Given** the wizard's explanation, **when** the human decides, **then** the install proceeds only on explicit confirmation; declining does not silently weaken the gate that required it.
- **Given** itd-62/spc-76's missing-scanner block, **when** it surfaces the prerequisite, **then** it routes through this wizard rather than a bare command.
- **Given** the explain-and-guide mechanics, **when** invoked outside Claude Code, **then** they run with no Claude-Code dependency.

## Open Questions

- Does the wizard ever RUN the install (confirmed, explained) or only show the step for the human to run? Running is friendlier; showing is safer/more portable.
- How does it describe a tool it does not have a canned blurb for — a curated registry of known deps vs a generated description?
- Is it a standalone surface or a sub-mode other surfaces invoke (itd-62 first)?

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

### Linkage note (spc-83.5)

Ships as one of FOUR intents sharing spec
`spc-83-operator-surfaces-manifest-lockstep`. abcd represents "N intents, one
spec" as a bundle (`kind: bundle-member` + shared `bundle: spc-83-operator-surfaces`)
— the representation the doc_fidelity intent-resolution + spec-close preflight
require. Bundle member by delivery relationship, not a scope change. The grill/PRD
bypass for this ungrilled intent is handled via the grandfather fields
(`prd_grandfathered` for GR002; two-key `grandfathered` + `grandfathered_at_phase`
for GR001). Full record in the spec's process-exception note.

## References

- Originating context: the itd-62/spc-76 grill (2026-06-29) — "always block on a missing
  scanner; provide a setup wizard that guides install" rather than degrade to advisory.
- First consumer: [[itd-62-pluggable-safety-gate]] (the safety gate's missing-scanner path).
- Thesis tie: keeping human JUDGMENT the constraint requires the human to understand what
  they are consenting to, not just be handed a command.
