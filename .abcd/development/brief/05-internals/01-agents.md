# Agent Catalog

abcd hands every judgement call to a model the host already runs, and keeps the
deterministic half for itself. What that buys a user: the default install needs no
API key and no model configuration, and every judgement comes back as JSON a verb
validates, so a model that hallucinates a citation produces a refusal rather than
a record. What it costs: abcd cannot judge anything on its own, and a verb whose
agent was never dispatched has no fallback verdict to offer.

An agent here is a **markdown prompt file** under `agents/`, host-delegated by
design (adr-25). The host owns model choice, credentials and execution; abcd
assembles the input, states the output contract, and checks what comes back.

## What ships

Fifteen agent prompts ship in `agents/` today, in four groups:

- **Lifeboat and release synthesis**, each feeding one verb that validates its
  output under a cite-or-be-dropped rule: `principle-distiller`
  (`disembark principles`), `press-release-composer` (`disembark press-release`),
  `lifeboat-reviewer` (`disembark review`), `graveyard-interpreter`
  (`disembark graveyard`), and `release-changelog-composer` (`launch ship`),
  which writes both documents of a release cut in one payload, the changelog
  lines and the release page, and whose payload is refused whole rather than
  cite-or-be-dropped.
- **The intent auditor**, `intent-auditor`, which judges a shipped intent's
  promise against delivered reality (below).
- **Repo-workflow reviewers and researchers**, dispatched by a human rather than
  by a verb: `docs-currency-reviewer`, `ruthless-reviewer`, `security-reviewer`,
  and `sota-researcher`.
- **The cold-reading instrument**: the four position definitions and the ledger
  `scribe` (below).

Each declares its inputs and outputs as JSON, and the schemas are the core's
rather than the prompt's. The record families — the issue schema, admissions and
surprises, dispositions, and the cold-reading run and item contract — share one
package, `internal/core/issueschema`, deliberately: the verb that writes a record
and the gate that judges the committed tree have to agree on what a well-formed
record carries, and two hand-kept copies drift the moment one side gains a field.
So the cold-reading item contract does not live with cold reading; the reading
package imports it from there.

`agents/` also holds two plain docs, its README and its changelog, which carry no
agent frontmatter. Because the plugin manifest declares no agents key, the loader
globs the flat `agents/*.md` set and registers both as harness agents alongside
the real prompt files; iss-110 tracks the mis-registration.

## The design roster still to be built

The lifeboat pipeline is drawn around a larger roster than the one that ships.
The rest are **design targets**, sequenced with
[Phase 6](../../roadmap/phases/phase-6-lifeboat.md), and none of them exists in
`agents/`:

| Agent | Pass | What it would produce |
|---|---|---|
| `flow-essence` | A | the spec spine, newest-first, with superseded decisions kept |
| `decision-archaeologist` | A | a decisions timeline from ADRs, conventions and git log |
| `review-collator` | A | consolidated reviews plus candidate pitfalls extracted from them |
| `code-rescuer` | A | code-level principles from a spec-window file selection |
| `chat-distiller` | B | rationale fills, unrecorded decisions and pitfalls from a time-windowed transcript subset |
| `artefact-curator` | C | an asset manifest classifying each item keep / adapt / drop, plus the lifeboat's docs copies |
| `brief-composer` | C | the lifeboat brief synthesising every Pass A/B/C input |
| `issue-scout` | C (opt-in) | issue entries annotated with related upstream work; prefers a peer scout over MCP when one is present, and is disabled by default |
| `embark-scaffolder` | embark | a scaffold plan for a target repo |
| `launch-gatekeeper` | launch | a release preflight over scan results and the payload manifest (itd-65, adr-33) |
| `documentation-auditor` | subagent | a documentation audit over a source or lifeboat `docs/` tree, invoked by other verbs rather than by a user |
| `reflection-composer` | reflect | the five retrospective sections, from a phase-audit receipt (itd-24) |

## The cold-reading definitions

The four `cold-reading-*` prompts are the reading positions of the cold-reading
instrument (itd-184 / spc-62), dispatched by the host over the input
`abcd reading assemble` produces
([`04-surfaces/23-reading.md`](../04-surfaces/23-reading.md)).

Each carries two frontmatter fields no other prompt has: the `position` it reads
at, and the `regime` that position reads under. The binary reads both. Bare `abcd
reading` names the definitions it resolves, and `reading ingest` resolves the
run's position to its definition, recomputes the definition's hash against what
the reading's output claims, and takes the regime from the definition rather than
from any operand or configuration key, so an output claiming a different regime is
refused.

A definition holds five parts, and a test asserts exactly that composition:
`## Object`, `## Question`, `## The blindness core`, `## Regime`, and
`## Item shape`. The repository sources the assembler admits at that position are
stated inside `## Object` rather than standing as a part of their own. The
blindness core is byte-identical across all four — a test holds it so — and states
what is true of every reading whatever its position; a definition that edited its
own copy would be claiming a licence its position does not hold. `## Item shape`
is what the ingest contract validates against: its body fields are read out of the
schema `reading ingest` uses, so a definition and the contract cannot drift, and
it carries exactly one fenced JSON block.

The sources named in `## Object` are what the position **may** read; the bundle
states what **this** run was given, and where the two disagree the bundle governs.

## `intent-auditor`: one agent, three roles

The auditor is one agent with three roles, sharing its prompt scaffolding, oracle
resolution and receipts. Each role has its own verb, so there is no dispatch by
record kind. Only the first ships.

1. **Single-document fidelity → `abcd intent audit <itd-N>`** (shipped). It reads
   a shipped intent's acceptance criteria and the delivered reality, and returns
   per-criterion verdicts plus a three-bucket prose audit (honoured / diverged /
   missing), each claim carrying a cited evidence pointer. The acceptance pass
   writes its verdicts into the intent's own `## Audit Notes`, which is the verdict
   of record; the per-run artefact beside it is an ephemeral review request in the
   gitignored local tier. The verb name is `audit` per adr-40, which reserves it
   for promise-versus-reality verdicts; the top-level `/abcd:audit` stays reserved
   for itd-16's hash-chain surface. Running it is manual: closing a spec moves the
   linked intent to `shipped/`, but nothing fires the auditor off that transition
   (spc-6 disowned auto-firing, and no spec owns it now). The term-drift,
   PRD-fidelity and modification-grammar outputs the role was drawn with are
   **deferred**: none is in the shipped prompt or in any lint.
2. **Cross-document fidelity → `abcd intent consistency [<itd-N>]`** (shipped,
   itd-48). It reads the brief and every intent outside `superseded/` and reports
   terminology drift, premise contradictions, scope leakage, sequencing
   impossibilities and naming conflicts, each finding naming two documents and
   quoting both verbatim. The binary assembles the corpus and validates the
   return on the audit's request/ingest seam; the ingest files each finding in the
   ledger, or links it to the open record that already holds it, and leaves a
   dated report on the reviews shelf. Per adr-40 the surface as first drawn was
   multi-act, its categories spanning `lint` and `audit`: what ships is the
   `audit` half alone, and the mechanical categories (schema and state
   contradictions, reference rot, acknowledgement gaps) are deferred to a
   follow-up intent.
3. **Kind classification → `abcd intent shape`** (**design target**). It would read
   the intent corpus and suggest reclassifications, supersessions and bundles. No
   `shape` sub-verb is registered, and no cached suggestions exist for bare
   `abcd intent` to surface.

The three roles share **one** catalogue entry: the roster grows by user-facing
responsibility, not by role.

## Oracle backend resolution

**Scope, per adr-25:** an agent that needs a model reaches it through the `oracle`
seam ([`02-adapters.md`](02-adapters.md)). The default is **host-delegated**: abcd
does the deterministic work and hands a prompt to the host's subagent dispatch,
which owns model choice, credentials and execution, and abcd consumes the
structured result.

Concrete backends are **opt-in adapters** behind the same seam, selected when an
operator wants abcd to reach a model directly: a local model, a model CLI run as a
subprocess, a provider API, or a model over MCP. The `oracle.backend` config key
records the choice, defaulting to host-delegated; an unreachable adapter degrades
to that default rather than blocking. Per
[`04-universal-patterns.md § 7`](04-universal-patterns.md#7-vendor-agnostic-adapters-with-environment-branching)
the seam is one interface with a native default and opt-in shapes, never a fixed
cascade the core imposes.

**Adapter guidance for high-stakes reviews (adr-25).** When an operator wires two
adapters, a **scoped** reviewer seeing only a selection and a **broad** reviewer
reasoning over the whole repo have complementary blind spots and are trusted
asymmetrically: the scoped verdict gates, the broad reviewer is mined for
findings, and the review-fix loop declares its stopping rule up front. That is
advice the adapter layer offers rather than a pipeline the core imposes.

## Verdict-tag protocol

Two verdict enums are in play, and they are deliberately disjoint.

**1. Review verdicts** — `{SHIP, NEEDS_WORK, MAJOR_RETHINK}` — assess a *change*.
They are emitted as tags in oracle output and are a shared convention at the
`spec`/`run` seam boundary (adr-24, adr-26), so any abcd-produced review is
portable across the seam without a tool-specific validator.

**2. Per-criterion acceptance verdicts** — `{MET, MET_WITH_CONCERNS, NOT_MET,
INCONCLUSIVE}` — assess a *promise against reality*, per itd-1. Rollup logic lives
in [`04-surfaces/05-intent.md`](../04-surfaces/05-intent.md).

Reviews emit family 1; auditors emit family 2. The two are never mixed.

## The scribe protocol

The `scribe` is machine assistance in maintaining the ledger, and its access rule
is the exact inverse of the assembler's (invariant 15 in
[`02-constraints/03-invariants.md`](../02-constraints/03-invariants.md), which
binds this section). The assembler passes a reading a positively included slice of
the shipped repository and no ledger; the scribe receives ledger content, the
run's reading records among it as the store holds them, plus the researcher's
supplied dispositions, and never the shipped repository as an object of
judgement. **No session holds both a reading and the ledger.** The scribe is
also not a consumer of the session-transcript store: that store's consumer list is
enumerated in the same invariant, and adding the scribe to it is an invariant
change rather than a code path.

The scribe's inputs block is an allow list rather than a deny list, because
positive inclusion is what excludes the path nobody thought to name, including a
record type the list has never heard of. Two tests in `internal/core/lint` hold the
definition to that, and their reach is exactly what they say: they prove the
definition names the right paths, not that a host assembled the right context.
Mechanical assembly belongs to `abcd scribe assemble`, which builds the context
from an allow list derived from the ledger's own directory list and parks it with
a manifest of every path passed; a third test holds the definition's list to
that function ([`04-surfaces/32-scribe.md`](../04-surfaces/32-scribe.md)).

The mechanical path exists beside the scribe: `abcd reading ingest` validates the
output a reading returned and writes its reading records, `abcd capture
disposition` writes the researcher's answer to one item, and `abcd scribe ingest`
validates what a scribe session returned, refuses anything it authored, and
writes it through the capture verbs' own functions. The scribe is the
transcription assistant of the session in which that material is prepared, and
four rules bind that session:

1. **Entries are transcribed when the reading returns**, not later. A protocol
   invented under time pressure is a protocol that gets skipped, and a batch of
   readings held for transcription is the pressure that invents one.
2. **The reading run and the scribe run are separate host sessions**, always. Each
   is retained under its own session id, and the transcript store is what shows
   two distinct sessions. Every reading bundle and every scribe context carries
   a per-run context stamp, and `abcd history separation` names a retained
   transcript carrying the reading stamp and the scribe stamp of one run. The
   honest limit: the check reports what a host retained; it cannot enforce that
   the practice held, because the separation happens in the host before anything
   is retained, and where nothing stamped was retained it reports the property
   unobserved rather than held.
3. **The transcribed material is committed through the ordinary record path.** The
   reading and disposition stores are declared record families, so `record_schema`
   holds each record to its shape at the gate, and the writing verbs validate
   before they write. A record the scribe transcribed reaches the tree through a
   verb — `abcd scribe ingest`, beside `abcd reading ingest` and `abcd capture
   disposition` — never by a hand-placed file.
4. **A fidelity flag is carried to the researcher unresolved.** The scribe may flag
   an internal inconsistency in the material it is transcribing, because that is
   transcription fidelity rather than judgement. It may never propose a
   resolution. The flag is a named field beside the transcribed material, so it can
   be counted and answered rather than buried in prose.

Anything the scribe is explicitly asked to produce **beyond formatting** opens
with a contribution stamp that travels with the material if it is adopted, and an
unstamped contribution is never delivered — a refusal in the definition, not a
preference. The stamp is the hand-run form of the record's origin and
production-mode keys (itd-178), which the writing verbs stamp on every record they
mint.

## Agent prompt frontmatter

Every prompt carries declared frontmatter. The fields:

| Field | Required | Purpose |
|---|---|---|
| `name` | yes | The agent's registered name; the flat-glob harness registration runs on `name` and `description` |
| `description` | yes | When the host should dispatch this agent |
| `color` | optional | A presentation hint. Nine prompts carry one: the four cold-reading definitions (cyan), `docs-currency-reviewer` (blue), `ruthless-reviewer` (orange), `security-reviewer` (red), `sota-researcher` (purple), and `intent-auditor` (green). It tracks no group — the auditor carries one and `lifeboat-reviewer` does not — so it is decoration a prompt opts into, not a signal to read |
| `tools` / `model` | optional | Tool allow-list and model hint for the host's dispatch, carried by the repo-workflow reviewer and researcher prompts |
| `prompt_version` | yes | Semver of the prompt, bumped on any prompt change; the changelog entry is keyed on it |
| `capability_scope` | yes | `{ task_classes: [...], designed_for: "<one line>" }`: the task classes the agent is designed for. `task_classes` is authored as a YAML inline list, never a block list, because the frontmatter parser does not support one nested there |
| `position` | cold-reading definitions | The reading position this definition is for; the locator resolves a position to its definition by this field |
| `regime` | cold-reading definitions | The supply regime that position reads under; `reading ingest` takes it from here and refuses an output whose own claim differs |
| `reads_untrusted_input` | yes | Whether the agent reads attacker-influenceable input: transcripts, lifeboats, forge issues, commit messages, model-emitted reviews |

A shipped check reads these. Record-lint's `agent_contract` rule, armed at blocker
severity over `agents/`, requires the declaration on every prompt whatever its
value — a rule that fired only on `true` is one a prompt opts out of by deleting a
line — and, on a prompt declaring `true`, both `capability_scope` fields plus an
injection-canary fixture that is present, a regular file and non-empty. An empty
file or a symlink is refused, because a canary that asserts nothing reports the
contract met without testing it. Every shipped prompt declares
`reads_untrusted_input: true` and carries a canary.

What is still a design target is the narrower half: set-membership of
`task_classes` against a closed enum. No enum file and no owning schema package
exists, so a token outside the intended set passes the gate. The reserved-
vocabulary table in
[`02-constraints/04-naming.md`](../02-constraints/04-naming.md) is the token set's
source of truth today, PR-to-extend (iss-265).

**Deliberately omitted** from agent frontmatter, as a boundary against scope
creep: runtime-appended failure modes, per-task-class model history, and
plan-time capability gating output. Those belong to the later-phase Frontier
Awareness intent.

**Why `capability_scope` rides in itd-5 rather than earning its own discipline:**
it is the same artefact class as `prompt_version` — agent frontmatter, versioned
with the prompt, mechanical to write — and its validation stays mechanical. The
linter never reads `designed_for` prose to judge scope, in either direction.
Anything fuzzier is the later-phase Frontier Awareness sub-check.

**The oracle seam contract is unchanged by it.** Capability-aware routing, when it
ships, is a pre-dispatch selector layer *above* the seam rather than a
modification to it: the selector consumes task class, agent and model to pick a
backend; the seam consumes a backend to dispatch. Thin seam.
