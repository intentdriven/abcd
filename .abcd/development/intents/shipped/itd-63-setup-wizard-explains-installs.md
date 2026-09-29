---
id: itd-63
slug: setup-wizard-explains-installs
spec_id: spc-2609211955339422
kind: standalone
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
impact: additive
---

# The Amateur Coder Is Told What Is Being Installed And Why, Not Just Asked To Run A Command

## Press Release

> **abcd gains a setup wizard: when a capability offers an optional adapter the amateur has not installed — or genuinely needs an external tool (a security scanner, a runtime, a CLI) — abcd explains in plain language WHAT the adapter or tool is, WHY the capability would use it, what its native default already covers, and what the tool does — then guides any install — instead of dumping a command the product thinker cannot evaluate.** Most bundled dependencies are optional adapters with native defaults ([adr-22](../../decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md)), so the wizard's first job is to explain that a capability already works on its native default and an adapter is an upgrade, not a prerequisite. Where a capability does have a hard external prerequisite — abcd's safety gate always blocks on a missing scanner — the thesis keeps human judgment the constraint, but a human cannot judge "install a scanner" if they do not know what it is or why their safety gate needs it. The wizard turns an opaque prerequisite into an informed choice: it names the tool, states the capability that requires it (e.g. "the security gate needs this to scan your app for vulnerabilities"), links what it is, shows the exact install step, and confirms before proceeding.

> "I'm fine installing things — I'm not fine installing things I don't understand," said a product thinker setting up abcd's safety gate. "Tell me this is a security scanner, that my safety check can't run without it, and what it'll do. Then I'll say yes. Don't just throw a command at me and assume I know."

## Why This Matters

abcd's safety gate (itd-62/spc-76) is designed to ALWAYS block on a missing scanner rather than degrade to advisory — the right call for the guarantee, but it puts an install prerequisite in front of an amateur who may not recognise the tool. That gate is a draft, not shipped: no gate on the default branch blocks on a missing scanner today, and the first shipped consumer of the explain-then-install mode is the history store's refusal when a repository has armed gitleaks for transcript capture. The thesis says keep the human's judgment the constraint; an install prompt the human cannot evaluate is judgment removed, not preserved. A setup wizard restores it: by explaining what and why, it lets the product thinker make an INFORMED decision rather than a blind one. This is a general need. Most abcd capabilities run on a native default and expose an optional adapter (adr-22) — the wizard explains that the native default already works and names what the adapter would add, so an optional install is never mistaken for a requirement. The few capabilities with a genuine external prerequisite (a security scanner, a runtime, a CLI, a model) get the same explain-then-install surface; itd-62 is the first caller.

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

## Mechanism

We expect people to accept an install they understand and decline one they do not, so an explanation before the ask raises the opt-in to the adapters that make a capability better; shown wrong if the same installs are still skipped, or blindly run, after it ships.

## Acceptance Criteria

- **Given** a verb finds a tool missing, **when** it names the gap, **then** it states the tool's name, whether it is optional or required for this capability, what works without it, what the tool would do, and the exact install step, from abcd's registry entry for that tool.
- **Given** the explanation, **when** the person answers, **then** the install runs only on an explicit yes and the verb reports what it ran and whether it worked; a no leaves the capability on its native default and says so, weakening nothing silently.
- **Given** a tool the registry does not know, **when** the gap is named, **then** the generic explanation and the step are shown and the registry gap is captured.
- **Given** the safety gate's missing-scanner case, **when** it surfaces the prerequisite, **then** it routes through this mode, not a bare command.
- **Given** a plain terminal outside any agent host, **when** the mode runs, **then** it works with no host dependency.

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that gave this intent its spec:

1. **Runs the install on confirmation**, and reports the result.
2. **Descriptions come from a curated registry** abcd ships; a gap is captured.
3. **A mode other verbs call**, not a surface of its own; standalone kind, since its bundle mate (itd-62) is a draft.

## Open Questions

_None open; decisions 1 to 3 settle the three this record carried._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-3c9fb4ba9770 -->
Fidelity review — receipt rcp-3c9fb4ba9770 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:02987ee70399625e2472201d6961c35bff8f37408256c2f78915e8785d534a0e
Input attestations: tree:ceb4b6dbb97622bddf2401c91f9f808ed05060a0 (origin/main; commits 0289455eb, edef56854, 705b8216d, 291410bb5, 09938d677, 7f34670d8 under internal/core/tools)@-;

Acceptance rollup: MET 2 · MET_WITH_CONCERNS 2 · NOT_MET 1 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: Explain renders name, optional/required per (tool, capability), what works without it, what it does and the platform step from the compiled registry; TestExplainNamesEveryPartTheCriterionAsksFor and the ahoy gap detector both exercise it, and the ahoy gap and the missing-gitleaks/gh refusals carry the lines
  evidence: internal/core/tools/explain.go:124 — "func (e Explanation) Lines() []string"
  evidence: internal/core/tools/explain.go:133 — "e.Tool + " — " + string(e.Requirement) + " for " + e.CapabilityName"
  evidence: internal/core/tools/registry.go:118 — "var registry = map[string]Tool{"
  evidence: internal/core/ahoy/detect.go:231 — "e := tools.Explain("gitleaks", capability)"
  evidence: internal/core/tools/tools_test.go:17 — "func TestExplainNamesEveryPartTheCriterionAsksFor"
- ac-2 — MET_WITH_CONCERNS: Install runs the step only after Confirm returns yes (nil or no or CI or --yes all decline), Result.Summary reports what ran and whether it verified, and every declined path ends in OnDecline (continuing on the native secret scanner); concern: the install is offered for gitleaks alone (DependencyTools), so for gh the explanation is shown in the refusal but no verb ever asks the question or runs the step
  evidence: internal/core/tools/install.go:179 — "ans := confirm(e) if !ans.Yes {"
  evidence: internal/core/tools/install.go:61 — "func (r Result) Summary() string"
  evidence: internal/core/ahoy/apply.go:465 — "res := newToolInstaller(a.cwd).Install(g.Tool.Tool, g.Tool.Capability, a.confirmTool)"
  evidence: internal/surface/cli/cli.go:3711 — "--yes never installs a tool"
  evidence: internal/core/ahoy/detect.go:207 — "var DependencyTools = []string{"gitleaks"}"
  evidence: internal/core/ahoy/tools_route_test.go:122 — "func TestDependencyNoKeepsTheNativeDefaultAndSaysSo"
- ac-3 — MET_WITH_CONCERNS: an unregistered name gets the generic text, no step (install step: none known to abcd) and a RegistryGap line naming the abcd capture that records it; concern: abcd files no capture itself (the line is printed for whoever meets it), and every production caller passes a literal registry name, so the unknown path is reached only by tests
  evidence: internal/core/tools/explain.go:87 — "func unknown(name string, capability Capability) Explanation"
  evidence: internal/core/tools/explain.go:104 — "func gapCapture(name string, capability Capability) string"
  evidence: internal/core/tools/tools_test.go:67 — "func TestUnknownToolGetsGenericTextAndARegistryGap"
  evidence: internal/core/ahoy/tools_route_test.go:183 — "func TestEveryToolAhoyNamesIsRegistered"
- ac-4 — NOT_MET: promised: the safety gate's missing-scanner case routes through this mode; delivered: no safety gate exists on main (itd-62 is still in intents/drafts/), so no gate surfaces the prerequisite at all; the close note substitutes the history store's armed-gitleaks refusal, which does carry tools.Missing, but that is transcript capture, not the safety gate the criterion names
  evidence: .abcd/development/intents/drafts/itd-62-pluggable-safety-gate.md:1 — "itd-62-pluggable-safety-gate.md (drafts/)"
  evidence: .abcd/development/specs/closed/spc-2609211955339422-setup-wizard-explains-installs.md:62 — "belongs to itd-62, which is still a draft: no gate on the default branch always blocks on a missing scanner"
  evidence: internal/core/history/history.go:277 — "err = tools.Missing(err, "gitleaks", tools.TranscriptScanArmed)"
- ac-5 — MET: the package reads no terminal and writes no stdout; the CLI supplies the Confirm and asks on a tty itself, refusing piped or --yes answers, and TestToolConfirmAsksOnlyAtATerminal plus TestAhoyInstallNamedToolReachesTheStep run the whole path with no host present
  evidence: internal/core/tools/registry.go:9 — "The package has no transport // knowledge — it never reads a terminal and never writes to stdout"
  evidence: internal/surface/cli/cli.go:3705 — "func toolConfirm(p ahoy.Prompter, named map[string]bool, yes bool, w io.Writer) tools.Confirm"
  evidence: internal/surface/cli/ahoy_tool_confirm_test.go:25 — "func TestToolConfirmAsksOnlyAtATerminal"

Gap audit:
- honoured:
  - a curated registry abcd ships, with what the tool is, why the capability uses it, the native default and the exact step per platform (decision 2)
    evidence: internal/core/tools/registry.go:118 — "var registry = map[string]Tool{"
  - the install runs on confirmation and its result is reported (decision 1)
    evidence: internal/core/tools/install.go:144 — "func (in *Installer) Install(name string, capability Capability, confirm Confirm) Result"
  - a mode other verbs call, not a surface of its own (decision 3): ahoy detect/install, history capture, ahoy remote call it
    evidence: internal/core/ahoy/detect.go:231 — "tools.Explain("gitleaks", capability)"
    evidence: internal/core/ahoy/remote.go:458 — "return nil, tools.Missing("
  - honesty about what the install does to the machine and the network
    evidence: internal/core/tools/registry.go:109 — "func homebrewEffects(program string) string"
  - declining never weakens a gate: the armed-gitleaks refusal stands, and the install never runs in CI or inside the repository tree
    evidence: internal/core/tools/install.go:159 — "if reason, ci := cienv.Runner(in.Getenv); ci {"
    evidence: internal/core/tools/install.go:237 — "func (in *Installer) admit(name string) (string, error)"
  - the plugin page relays the explanation and the host-relayed yes through --install-tool
    evidence: commands/ahoy.md:179 — "**The tool question.**"
- diverged:
  - the install question is asked for every missing tool the mode explains: gh is explained in the ahoy remote / site setup refusal but never offered (DependencyTools holds gitleaks only, and --install-tool gh is refused)
    evidence: internal/core/ahoy/detect.go:207 — "var DependencyTools = []string{"gitleaks"}"
    evidence: internal/surface/cli/cli.go:3691 — "is not a tool ahoy install checks for"
  - a registry gap is captured: abcd prints the capture command for the person to run, it does not file the capture
    evidence: internal/core/tools/explain.go:104 — "record the gap in abcd's own ledger with: abcd capture"
  - the guard's and the launch's tool checks reroute through the mode (spec scope 3): neither exists, nothing was rerouted
    evidence: .abcd/development/specs/closed/spc-2609211955339422-setup-wizard-explains-installs.md:68 — "do not exist: the guard runs no external tool, and the launch scans are native"
- missing:
  - the safety gate's missing-scanner path routes through the mode (first consumer, itd-62): the gate is not on main
    evidence: .abcd/development/intents/drafts/itd-62-pluggable-safety-gate.md:1 — "drafts/"
<!-- abcd-review-end receipt=rcp-3c9fb4ba9770 -->

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
- Intended first consumer: [[itd-62-pluggable-safety-gate]] (the safety gate's missing-scanner path), still a draft; its missing-scanner path must call `tools.Missing` / `tools.Install` rather than print a bare command. The shipped consumers are the history store's armed-gitleaks refusal and ahoy's remote verbs' missing-`gh` refusal; both route through `tools.Missing`.
- Thesis tie: keeping human JUDGMENT the constraint requires the human to understand what
  they are consenting to, not just be handed a command.

## Grounds

- pursued: the run lands two new adapters and a scanner rule, each of which will meet a machine without the tool; we expect the explained ask to be answered yes where a bare command was ignored; shown wrong if the same installs are still skipped or blindly run
