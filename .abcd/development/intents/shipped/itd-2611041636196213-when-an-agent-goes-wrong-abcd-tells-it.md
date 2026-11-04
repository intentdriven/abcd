---
id: itd-2611041636196213
slug: when-an-agent-goes-wrong-abcd-tells-it
spec_id: spc-2611041649558539
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-3, itd-103]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Agents are corrected when wrong and steered where no check reaches

## Press Release

> When an agent goes wrong, abcd tells it why and what to do instead; where no check can catch the mistake, abcd steers it in a short line as that work starts. Alice, a solo founder, watches an agent try an unsafe delete: it is refused and shown the safe form, in a sub-agent as much as in the main session. The session she leaves running for days holds the work, not forty-five rules no agent read, and her next usage panel has one less thing on it.

## Why This Matters

abcd's rules exist for two jobs: to correct an agent that has gone wrong, and to
steer it towards what no check can catch. Today one mechanism does both, and does
neither well. The prompt hook sends every matching rule in advance. In the seven
days to 4 Nov 2026 the host cut that output 475 times to a 2,000-character preview
(iss-2611041419354106), and the session ledger recorded the cut text as delivered.
At tip, ordinary prompts produce 12,000 to 38,000 characters against a cap of
10,000, out of 72,002 characters in all.

Meanwhile the guard already does the first job well. Its refusal carries why and
what to run instead, at the exact command, and it reaches sub-agents, which never
receive the prompt hook. In the 30 days to 4 Nov 2026 it made 478 refusals in about
145,000 shell commands; 368 were in sub-agents, and in main sessions the latest
prompt named the command in only 11 of 110.

The research is thinner than the vendor limits suggest. McMillan (arXiv 2605.10039,
1,650 sessions) finds no detectable effect of file size from 25 to 500 lines, so no
count of rules is a proven ceiling; this intent rests on the delivery limit and the
logs, not on a number. A single-author test (github.com/B0yko/obey, Haiku 5.5, not
peer reviewed) saw a hook alone reach full compliance with the rule text removed. No
controlled study compares refusal alone with pre-teaching plus refusal.

The 129 rules of today are a snapshot. What lasts is the system: rules come and go,
and each says which job it does.

## Mechanism

We expect agents to behave as they do now on far less text, because the rule text
that is cut today changes nothing and the guard already corrects what it refuses. In
the 30 days to 4 Nov 2026 main sessions took 110 refusals. If that rate more than
doubles on comparable work once the text is gone, the claim is wrong and the most
refused hazards go back into the steering line.

## Scope Conditions

- Holds for a host that runs the pre-tool-use and prompt hooks, where the pre-tool-use <!-- cond: cond-2611041649555302 -->
  hook runs before the host's own permission prompt. A host with neither reads the same
  rules through `abcd rules`.
- The cap is the host's documented 10,000 characters for one hook's output, read <!-- cond: cond-2611041649558257 -->
  2026-11-04; another host's cap may differ, and the budget sits in one place.
- A correction helps only if it comes before the harm. A warned command still runs, so <!-- cond: cond-2611041649557746 -->
  its rule stays a steering rule, and so does a rule whose check fires too late to save
  the work, such as one at preflight.

## Acceptance Criteria

- **Given** a prompt, **when** the hook answers, **then** its whole output, rules, notes,
  status and away block together, is within the host's cap.
- **Given** matched rules that do not all fit, **when** the hook answers, **then** each
  domain goes in full or as its first rule and a line giving the command that shows the
  rest, and the session records only that: nothing the host cut is counted as sent.
- **Given** full rules and a matching note, **when** the hook answers, **then** the note
  still has room, and a note or page too long to fit comes back as one line rather than
  not at all.
- **Given** a prompt about shell work with the guard on, **when** the hook answers,
  **then** the agent gets one line saying a guard judges each command and the rules for
  commands the guard only warns about, none of the text for commands it refuses, and
  `abcd rules SHELL` still lists every hazard.
- **Given** a repository that has switched its guard off, **when** the hook answers,
  **then** every shell rule stays a steering rule, each opens `Hazard (guard off)` under a first
  line saying the guard is off, and the line saying a guard judges each command is not
  sent. Eleven registry entries keep
  wording that names the guard after that opening; rewording them belongs to the
  rule-by-rule change.
- **Given** the bundled registry, **when** its tests run, **then** every entry that refuses
  or warns says why and what to do instead, and every entry appears in exactly one of the
  two lists, so a hazard can never fall out of both.
- **Given** a person wants a new rule, **when** a machine can catch it in a shell command,
  **then** they add an entry to `.abcd/guard.json`, and otherwise a line to `rules.json`,
  neither of which changes format, so no existing override file has to change.
- **Given** `abcd rules --json`, **when** a reader asks what each domain sends, **then**
  it separates the steering rules from the ones a check carries and gives their sizes, so
  the review of the rule set can be run again as rules change.

## Open Questions

None. The product thinker confirmed the reversal below at sign-off, 2026-11-04.

## Decisions

- itd-103 is refined, not superseded: the registry, the refusal and `abcd rules SHELL`
  stay, and only what is sent ahead of time changes.
- itd-2609292109214516 stays separate: it makes injection a seam for another back end,
  and this intent changes what the native loader sends. Whichever lands second carries
  the budget and the two lists through its seam.
- A delete entry counts as refused though it sits in the warn tier: the guard refuses it
  in a repository unless it proves the target safe, so only what is sure to run is sent
  ahead of time. Its lesson keeps its `Warned by the guard` opening until the
  rule-by-rule change.
- Whatever of the 2,500 bytes kept for notes they leave unused goes back to the rules,
  so a short note does not shorten a domain that fits beside it.
- With the guard off, the rules cannot all fit one turn, so the first rule says the guard
  is off and the rest are read with `abcd rules SHELL`.
- Refusal counts per hazard are out of scope: the mechanism's threshold reads the logs
  already kept, so a counter waits for a check that needs it.

## Related

- **Reverses** the teaching plane of itd-103 as wired by the product thinker's rulings
  J10 (dcn-2609290000000014) and CK1 (dcn-2609300000000008): refused hazards are no longer
  taught ahead of time, as the product thinker confirmed on 2026-11-04.
- **Narrows** dcn-2610161612269647 on teaching the command forms the host can check: a
  form the guard refuses is corrected by the refusal, which comes before the host's own
  prompt, so the agent does not stall waiting for a person.
- **Refines** spc-2610111627213138: its brackets and its shortened form are kept, and the
  same shortened form now also answers an overflow; rules still come back in full whenever
  they fit, so its ruling stands.
- This is the whole delivery: the same change resolves iss-2611041419354106 and carries
  `Delivers:` for this intent.
- Applying the rule-by-rule review of the snapshot is a later change, once the product
  thinker has marked up the review page; nothing here rewords a rule.

## Audit Notes

<!-- abcd-review: OWED receipt=rcp-889ac8de461c -->
Fidelity review OWED (receipt rcp-889ac8de461c).
<!-- abcd-review-end receipt=rcp-889ac8de461c -->

## Grounds

- pursued: agents behave as they do now on far less text, since text the host cuts changes nothing and the guard already corrects what it refuses; shown wrong if main-session refusals more than double from 110 per 30 days on comparable work
