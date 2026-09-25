---
name: ideate
description: "Judge an idea through the host-run admission gauntlet: Writes nothing bare, and one research record and its decision-log line; refuses an unknown sub-verb."
argument-hint: "<the idea, in one or two sentences>"
block: agents
---

# `/abcd:ideate` — the idea-admission gauntlet

An exciting idea is cheapest to kill in its first hour and most expensive to kill
after it has minted intents, specs, and branches. This runs the admission
interview and records the outcome.

**Ideate is optional and never a gate.** Nothing requires it: `/abcd:capture` and
`/abcd:intent` work exactly as they always do, and skipping ideate is not a
mistake to warn about. Reach for it when an idea is big and unproven enough that
being wrong about it would be expensive.

**The binary does no judging.** The three legs below are yours to run. `abcd
ideate record` validates what they produced, proves every citation names a real
record, and writes the durable verdict. It never fetches a source and never
forms an opinion.

## The three legs, in order

The order is not cosmetic — each leg changes what the next one is looking at —
and `abcd ideate record` refuses a payload whose legs are missing or out of
order.

### Leg 1 — Primary-source research

Research the idea's **load-bearing claims**: the ones that, if false, take the
idea with them. For each, open the **primary** source — the paper, the
specification, the API reference, the original data — and check the claim
against it.

A secondary citation is not a check. In the run this protocol comes from, three
secondary-source claims were falsified once the primary was opened, one of them
an extraction artefact that no source ever said.

Produce, for each claim: the claim, the primary source (a URL or a document
reference), and the finding — `verified`, `falsified`, or `unverifiable`. An
honest `unverifiable` is a result; a missing source is not, and the binary
refuses a claim that names none.

### Leg 2 — Record grill

Read the existing record before anything else is decided — the brief, the
intents in **every** bucket (`drafts/`, `planned/`, `shipped/`, `disciplines/`,
`superseded/`), the ADRs, the principles, the research notes under
`.abcd/development/research/notes/`, and the decision log
`.abcd/work/DECISIONS.md`. Answer one question: does an entry already
**cover**, **contradict**, or **supersede** this idea?

Every hit is **cited by record id** (`itd-N`, `spc-N`, `iss-N`, `adr-N`), and
`abcd ideate record` refuses the whole verdict if an id does not resolve in this
repository. Check the id before you write it down. The brief, the principles,
the research notes, and the decision log carry no per-entry id, so a hit that
rests on one goes in the `note` field of the nearest citable record. A standing
verdict on the submitted idea — recorded in a research note or a decision
line — is precisely the hit this leg exists to surface; missing it re-opens a
decided question.

Finding **nothing** is a legitimate and common outcome — say so; do not
manufacture a hit.

### Leg 3 — Adversarial review

Hand the idea to a **fresh evaluator whose only job is to kill it**. Three
properties make this leg worth running, and they are the whole point of it:

- **Fresh context.** The evaluator must not be the session that ran legs 1 and 2.
  Dispatch it as a separate agent with its own context.
- **Off-policy.** Do not hand it the framing, the enthusiasm, or the conclusions
  of the earlier legs. It receives the idea and the two legs' *findings*, not
  their narrative.
- **Unknown authorship.** **Strip every authorship signal before handing the
  artefact over**: no "the user's idea", no "our proposal", no session history,
  no mention of who wants this or how excited they are. The evaluator sees an
  artefact of unknown provenance. This is the only measured debiasing effect in
  the whole protocol; leaking authorship converts an adversary into a colleague.

Ask it for concrete **kill attempts**, each with an outcome: `survived`,
`partial` (it landed but is not on its own fatal), or `fatal`.

## Record the verdict

Compose the verdict document and hand it to the binary:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" ideate record <idea-slug> --verdict-json <path>   # or - for stdin
```

`<idea-slug>` is lower-case kebab-case (it becomes a filename). The document:

```json
{
  "schema_version": 1,
  "prompt_version": "1.0.0",
  "idea": "the idea in the words it entered with",
  "legs": [
    {"kind": "research", "claims": [
      {"claim": "…", "primary_source": "https://…", "status": "verified|falsified|unverifiable"}
    ]},
    {"kind": "record-grill", "hits": [
      {"record": "itd-104", "relation": "covered|contradicted|superseded", "note": "optional"}
    ]},
    {"kind": "adversarial-review", "kill_attempts": [
      {"attempt": "…", "outcome": "survived|partial|fatal"}
    ]}
  ],
  "verdict": "survives|killed|reframed",
  "rejected_alternatives": [
    {"alternative": "…", "why_rejected": "…"}
  ]
}
```

**The verdict is recorded either way.** A killed idea earns exactly as full a
record as a surviving one — that record is what stops the idea being talked back
into six months from now.

**Rejected alternatives are not optional.** If genuinely none were considered,
say so explicitly with `"no_rejected_alternatives": true` and an empty list; the
binary refuses an empty list that arrives by omission, because silence and "we
weighed nothing" are indistinguishable to a later reader.

## What the binary does with it

It writes `.abcd/development/research/notes/<date>-ideate-<slug>.md` — the idea, the
three legs, the verdict, and the rejected alternatives, rendered for a human —
and appends one dated pointer line to `.abcd/work/DECISIONS.md`.

**Both paths are relative to the checkout root, from anywhere in the tree.** The
verb resolves the repository root before it validates anything, so the grill
reads the checkout's own record — a hit citing `itd-104` resolves from a package
directory exactly as it does from the root — and both artefacts land in the
checkout's store. Outside a repository there is no research store to write into,
and the verb refuses (exit 2) rather than laying one where it stands: a verdict
filed outside every checkout reaches no gate and no reader, and a killed idea
nobody can find is an idea that gets proposed again. If a research store also
exists below the repository root, the verb names it on stderr and leaves it
alone; relay that line.

Both are committed, so every free-text field of the verdict is scanned and
redacted before either is written: a token, an email, or an absolute home path in
the idea, a claim, a note, a kill attempt, or a rejected alternative is rewritten
to a placeholder, and the count comes back as `redactions`. A repository whose
scanner config cannot be read refuses the run outright rather than record under a
detector it cannot trust — nothing is written, and the same payload records once
the config is repaired.

Exit codes:

- **0** — recorded. Report the `verdict`, the `path`, and the `cited_records`
  (the ids the grill cited, every one proved to resolve). Show the user the
  written record. If `redactions` is non-zero, say so — the record no longer
  matches what was composed, and whoever composed it needs to know a credential
  or an identity was in the text.
- **2** — refused, and **nothing was written**. Relay the diagnostic: an
  unresolvable citation names the id, a payload fault names the field, a
  verdict record that already exists under today's date is refused rather than
  overwritten (give the re-run its own slug, or keep the first record), and a
  working directory with no repository above it — or one git will not answer
  for — is refused because there is no research store to address.

## After a verdict

- **survives** — the idea may graduate to a draft intent through the ordinary
  path, `abcd intent "<text>"`. Ideate mints no intent itself; graduation stays a
  deliberate act you take with the user.
- **killed** — stop. Do not file an intent, and do not soften the record.
- **reframed** — any graduation carries the reframing recorded in leg 3, not the
  original wording.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
