---
name: ideate
description: "Judge an idea through the host-run admission gauntlet: Writes nothing bare, and one verdict record, private or public; refuses an unknown sub-verb."
argument-hint: "<the idea, in one or two sentences>"
block: people
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
ideate record` validates what they produced, proves every record id the legs
cite names a real record, and writes the durable verdict. It never fetches a source and
never forms an opinion.

## Private unless the person says public

Every idea is **private** unless the person has said it is **public**. Do not
ask: record it private, and pass `--public` only when the person has said in
so many words that the idea is public, never on an inference from the idea or
the repository. The choice decides who can ever read the verdict.

- **Private** (the default) — the verdict stays on this machine, in
  `~/.abcd.noindex/ideas/<root-sha>/`, readable by the person's account alone,
  and no decision points at it. Name no source behind the idea anywhere
  public while it is private.
- **Public** (only on the person's word) — the verdict is written into the repository, under
  `.abcd/development/research/ideas/<root-sha>/`, with a decision record in
  `.abcd/work/decisions/` pointing at it, and is public once pushed, sources
  included.

`<root-sha>` is the repository's root commit, which the binary resolves itself.

## Where the idea came from

Before the first leg, note the sources the idea came from — the post, the paper,
the conversation — as the verdict's `origin_sources`. An idea with nothing behind
it but its own words takes an empty list; the list is never left out, because
leg 1 is checked against it.

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

Produce, for each claim: the claim, the primary source, and the finding —
`verified`, `quoted`, `falsified`, or `unverifiable`. A primary source is one of
three things: an `https://` address, `source:<key>` for a source filed in your
sources corpus (`/abcd:library`), or a record id. Anything else — a remark heard
in passing, a local file — is refused. Where a sources corpus exists, a
`source:<key>` must name an entry in it.

A claim checked only against the source the idea came from is `quoted`: the
source says what the idea says it says, which proves the quotation, not the
claim. The binary refuses a claim marked `verified` whose primary source is one of
the `origin_sources`. An honest `unverifiable` is a result; a missing source is
not, and the binary refuses a claim that names none.

### Leg 2 — Record grill

Read the existing record before anything else is decided — the brief, the
intents in **every** bucket (`drafts/`, `planned/`, `shipped/`, `disciplines/`,
`superseded/`), the ADRs, the principles, the research notes under
`.abcd/development/research/notes/`, the public verdicts under
`.abcd/development/research/ideas/`, and the decisions in
`.abcd/work/decisions/` (`abcd decide` lists them, newest first), with a
repository's frozen `.abcd/work/DECISIONS.md` where one remains from before
the folder. Read this machine's private verdicts in
`~/.abcd.noindex/ideas/<root-sha>/` too, but never quote one into a public
verdict. Answer one question: does an entry already
**cover**, **contradict**, or **supersede** this idea?

Every hit is **cited by record id** (`itd-N`, `spc-N`, `iss-N`, `adr-N`,
`dcn-N`), and `abcd ideate record` refuses the whole verdict if an id does not
resolve in this repository — in a hit, and in a claim or its primary source, a
note, a kill attempt, a reviewer's label or sources, the ruling, a rejected
alternative, an origin source or the reframing (an id inside a web address is
part of the address). A search term naming an id is not checked, because
searching for one is the finding. Check the id before you write it down. The brief, the principles,
the research notes, and a frozen log's lines carry no per-entry id, so a hit
that rests on one goes in the `note` field of the nearest citable record. A
standing verdict on the submitted idea — recorded in a research note or a
decision — is precisely the hit this leg exists to surface; missing it re-opens a
decided question.

Record **every search** you ran: the terms, the places you searched, and how many
entries came back. Finding **nothing** is a legitimate and common outcome — say
so; do not manufacture a hit. A grill with no hits must carry its searches, and
the binary refuses one that carries neither, because an empty grill and a broken
search would otherwise write the same record.

### Leg 3 — Adversarial review

Hand the idea to **fresh evaluators whose only job is to kill it** — two
reviewers, each its own agent. Four properties make this leg worth running, and
they are the whole point of it:

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
- **One idea per context.** Give each reviewer exactly one idea in its context,
  and dispatch a new reviewer for the next idea. A reviewer that has already
  judged another idea compares the two instead of attacking this one.

Ask each reviewer for concrete **kill attempts**, each with an outcome:
`survived`, `partial` (it landed but is not on its own fatal), or `fatal`; for its
own verdict (`survives`, `killed` or `reframed`); and for the sources it
consulted, in the same three shapes as a primary source. Record each reviewer
under a label of its own. If the reviewers' verdicts differ, settle the split and
record the **ruling**: who ruled, the verdict they reached, and why. The binary
refuses a split with no ruling.

## Record the verdict

Compose the verdict document and hand it to the binary. With no flag, or with
`--private`, the verdict is recorded privately; `--public` records it in the
repository and is passed only on the person's word. Both flags together are
refused, writing nothing.

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" ideate record <idea-slug> --verdict-json <path>   # private; add --public on the person's word; - for stdin
```

`<idea-slug>` is lower-case kebab-case (it becomes a filename). The document:

```json
{
  "schema_version": 2,
  "prompt_version": "1.0.0",
  "idea": "the idea in the words it entered with",
  "origin_sources": ["https://… the post the idea came from"],
  "legs": [
    {"kind": "research", "claims": [
      {"claim": "…", "primary_source": "https://…", "status": "verified|quoted|falsified|unverifiable"}
    ]},
    {"kind": "record-grill",
     "hits": [
       {"record": "itd-104", "relation": "covered|contradicted|superseded", "note": "optional"}
     ],
     "searches": [
       {"terms": ["…"], "places": [".abcd/development/intents/"], "count": 1}
     ]},
    {"kind": "adversarial-review",
     "reviewers": [
       {"label": "reviewer-a", "verdict": "survives|killed|reframed",
        "kill_attempts": [{"attempt": "…", "outcome": "survived|partial|fatal"}],
        "sources": ["https://…"]},
       {"label": "reviewer-b", "verdict": "survives|killed|reframed",
        "kill_attempts": [{"attempt": "…", "outcome": "survived|partial|fatal"}],
        "sources": []}
     ],
     "ruling": {"by": "who ruled", "verdict": "survives|killed|reframed", "reason": "…"}}
  ],
  "verdict": "survives|killed|reframed",
  "reframing": "only with a reframed verdict: the shape of the idea that survives",
  "rejected_alternatives": [
    {"alternative": "…", "why_rejected": "…"}
  ]
}
```

**The verdict is recorded either way.** A killed idea earns exactly as full a
record as a surviving one — that record is what stops the idea being talked back
into six months from now.

`ruling` is needed only when the reviewers' verdicts differ, and `reframing` only
with a `reframed` verdict, which the binary refuses without one; a reframing beside
any other verdict is refused too. The binary records version 2 only: a payload of
`"schema_version": 1` is refused with a message naming version 2.

**Rejected alternatives are not optional.** If genuinely none were considered,
say so explicitly with `"no_rejected_alternatives": true` and an empty list; the
binary refuses an empty list that arrives by omission, because silence and "we
weighed nothing" are indistinguishable to a later reader.

## What the binary does with it

It writes one record, `<date>-ideate-<slug>.md` — the idea and where it came
from, the three legs (with the grill's searches, each reviewer's section and any
ruling), the verdict, the rejected alternatives and any reframing, rendered for a
human — to the store the flags name:

| Flags | Record | Decision |
|---|---|---|
| none, or `--private` | `~/.abcd.noindex/ideas/<root-sha>/`, owner-only | none |
| `--public` | `.abcd/development/research/ideas/<root-sha>/` | one new record in `.abcd/work/decisions/`, pointing at the verdict |

A private record is refused when the home directory lies inside a git working
tree that is not the account's own, because a commit there could carry it.
Before writing one, the verb makes sure `~/.abcd.noindex/.gitignore` has been
laid, so a git repository in the home folder passes the record by; where that
file cannot be laid, the private record is refused and nothing is written. A
repository with no commit yet has no root commit to key either store on, and a
shallow clone names its graft in place of the root, so both are refused; fetch
the full history (`git fetch --unshallow`) and record again.

**The public paths are relative to the checkout root, from anywhere in the tree.** The
verb resolves the repository root before it validates anything, so the grill
reads the checkout's own record — a hit citing `itd-104` resolves from a package
directory exactly as it does from the root — and both artefacts land in the
checkout's store. Outside a repository there is no research store to write into,
and the verb refuses (exit 2) rather than laying one where it stands: a verdict
filed outside every checkout reaches no gate and no reader, and a killed idea
nobody can find is an idea that gets proposed again. If a research store also
exists between the working directory and the repository root, the verb names it on stderr and leaves it
alone; relay that line.

A public verdict is committed, and a private one is a file on disk, so in both
stores every free-text field of the verdict is scanned and redacted before
anything is written: a token, an email, or an absolute home path in
the idea, a source, a claim, a note, a search, a kill attempt, the ruling, a
rejected alternative, or the reframing is rewritten
to a placeholder, and the count comes back as `redactions`. A repository whose
scanner config cannot be read refuses the run outright rather than record under a
detector it cannot trust — nothing is written, and the same payload records once
the config is repaired.

Exit codes:

- **0** — recorded. Report the `visibility`, the `verdict`, the `path`, and the `cited_records`
  (the ids the grill cited, every one proved to resolve, as is every id cited
  elsewhere in the verdict, apart from the idea text and the grill's searches). Show the user the
  written record. If `redactions` is non-zero, say so — the record no longer
  matches what was composed, and whoever composed it needs to know a credential
  or an identity was in the text.
- **2** — refused, and **nothing was written**. Relay the diagnostic: a run
  with both `--private` and `--public` is refused (re-run with the one the
  person chose, or with neither for private), an
  unresolvable citation names the leg and the id, a primary source of no known
  shape, a `source:<key>` the corpus does not carry, an empty grill with no
  searches, a split between reviewers with no ruling, a reframed verdict with no
  reframing, a `verified` claim resting on the idea's own source, and a version 1
  payload are each refused naming what to fix, a payload fault names the field, a
  verdict record that already exists under today's date is refused rather than
  overwritten (give the re-run its own slug, or keep the first record), a
  public verdict in a repository with no shared tier (`.abcd/work/`) is
  refused because its decision has nowhere to land (run `ahoy install`), and a
  working directory with no repository above it — or one git will not answer
  for — is refused because there is no research store to address.

## After a verdict

- **survives** — the idea may graduate to a draft intent through the ordinary
  path, `abcd intent "<text>"`. Ideate mints no intent itself; graduation stays a
  deliberate act you take with the user.
- **killed** — stop. Do not file an intent, and do not soften the record.
- **reframed** — any graduation carries the verdict's `reframing`, recorded
  under "The reframing", not the original wording.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout. In a checkout the
person has listed in `~/.abcd.noindex/dogfood-roots`, or a worktree that
checkout records, `"${CLAUDE_PLUGIN_ROOT}/abcd" hook dogfood-binary` prints
the path of a build of the reviewed branch's tip, as the remote that line
declares last reported it, once it is ready: run that path in place of the
plugin-root binary, and the plugin-root binary while it prints nothing.

**User input:** $ARGUMENTS
