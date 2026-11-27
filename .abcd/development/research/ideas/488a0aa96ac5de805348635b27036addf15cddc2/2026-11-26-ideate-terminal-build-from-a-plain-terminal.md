# Ideate verdict — terminal-build-from-a-plain-terminal

**Verdict: reframed.** Recorded on 2026-11-26 by abcd's idea-admission protocol —
primary-source research, a grill against the existing record, and an
independent adversarial review. This record exists so the idea is not
re-litigated: it stands whether the idea lived or died.

## The idea

Run an intent's build entirely from a plain terminal, unattended: `abcd build itd-N` (or a drive) takes one READY intent to delivered, starting its own agents on a separate login, with no Claude Code session driving the loop — as `abcd drain --drive` already does for issues.

## Where the idea came from

- itd-2611251725209625, the draft intent filed on 2026-11-25 that already holds this idea, in nearly its words ("abcd build runs from a plain terminal, as abcd drain already does"). The gauntlet was first run without it; see the correction under leg 2.

## Leg 1 — Primary-source research

Every load-bearing claim checked against its primary source, never a
secondary citation. A claim marked quoted was checked only against the
source the idea came from, which proves the quotation and not the claim.

| Claim | Primary source | Finding |
|---|---|---|
| The host CLI runs non-interactively with -p, and an unattended run can deny every permission prompt (--permission-prompts none), so a process can start each stage's agent with no person at the keyboard. | https://code.claude.com/docs/en/headless | verified |
| A subscription-only person can be served without an API key: --bare never reads OAuth credentials or the keychain, so the subscription route must be a non-bare launch walled by its own flags; the doc says --bare will become the default for -p in a future release, which a walled non-bare launch must then opt out of. | https://code.claude.com/docs/en/headless | verified |
| A separate login can sit beside the person's own: CLAUDE_CONFIG_DIR overrides the configuration directory and is documented as useful for running multiple accounts side by side; ANTHROPIC_API_KEY, when present in -p, is always used over the subscription. | https://code.claude.com/docs/en/env-vars | verified |
| The vendor's terms permit the person's own unmodified CLI on their own subscription and bar a third party from routing requests through subscription credentials on users' behalf or storing their tokens; advertised Pro/Max/Ultra limits assume ordinary, individual usage of Claude Code and the Agent SDK, so a long unattended build spends against a limit sized for individual use. | https://code.claude.com/docs/en/legal-and-compliance | verified |
| abcd drain --drive already runs issue lanes from a plain terminal, person-typed, refused inside an agent session, with every stage's agent on a separate profile login behind an allow-list wall, a cap, a yes and a hand-back; its intent is still planned because its spec closed with a remainder, which the open spc-2611051628562376 holds as three steps: the live proof on the one walled login (MUST, live, not marked done), then the key route and a lane paused past an outage (both AFTER). | itd-2610222215485265 | verified |
| The no-host drive for abcd build is already designed and sequenced: step 12 of the build spec adopts adr-2610222222326104's four conditions for build, and is built only once the drain intent ships. | spc-2609202134338445 | verified |
| The drain intent cannot ship until the three steps of its remainder do: first the live proof of the drive on the one walled login, on the subscription route (MUST, live), then the key route, which waits on two draft intents (the API-key budget and named profiles), and the ruling on a lane paused past an outage; so the build's terminal drive is gated on a live proof still open as well as on a key route a subscription-only person does not use. | spc-2611051628562376 | verified |
| The drive today steps only the drain's issue runs: its set-aside and stuck-count handling of a refused receipt is kept on issue runs, and an intent's run keeps the stop. | https://github.com/intentdriven/abcd/blob/c89cfe633/commands/build.md | verified |
| A READY intent needs no human act mid-run: planning is done before READY, the landing closes the spec through the loop's own reconcile, and merge arming follows facilitator.mode (a human approval, or a recorded review pair in auto). | https://github.com/intentdriven/abcd/blob/c89cfe633/internal/core/implement/loop/land.go | verified |
| An unattended run cannot plan a draft: --auto-plan's two plan reviews run on the host's agents only, and acceptance criteria are a person's that nothing unattended writes; so the idea holds only for an intent already READY. | adr-2610310235000654 | verified |
| The host-stepped build's own claude runner still runs bare on an API key only, so a subscription user cannot route a build role to it today; whether the build runner takes the drive's subscription default is an open question to the product thinker. | iss-2610240932215918 | verified |
| Operating an unattended drive is the hard part, not starting it: the drain drive's three-day run landed twenty merged pull requests but took a second session watching it and six attempts over six hours to restart, and stalled 23 hours without saying so. | itd-2611141610197753 | verified |

## Leg 2 — Record grill

Does the brief, an intent, an ADR, or a principle already cover,
contradict, or supersede this idea? Every hit is cited by record id, and
every id resolved in this repository when the verdict was recorded.

| Record | Relation | Note |
|---|---|---|
| spc-2609202134338445 | covered | Step 12, 'The no-host drive', is this idea: the build loop driving itself with no host session under adr-2610222222326104's four conditions, widening AGENTS.md's host-delegated boundary to name it; blocked_by itd-2610222215485265, and a new run of the spec is refused at the build's blocked check until it ships. |
| itd-2609201916151817 | covered | The build intent (planned): its press release promises one intent from READY to delivered with nobody in the loop; decision 5 makes the process driver opt-in and a reversal flag on the host-delegated boundary, confirmed for the opt-in path only. |
| adr-2610222222326104 | covered | Accepted: no-host drives only on the person's own invocation, their own account behind a wall, bounded, handing back what they cannot finish; Consequences: 'abcd build can adopt the same four conditions later, under this record'. AGENTS.md Boundaries names drain --drive as the one written exception. |
| itd-2610222215485265 | covered | The drain analogue the idea cites, and the blocker of build step 12; planned, its spec spc-2610230813348813 closed with a remainder that the open spc-2611051628562376 holds as three steps: the live proof on the one walled login (MUST, live, still open), the key route (AFTER, waiting on itd-2611011230226453 and itd-2611040344170206) and the ruling on a lane paused past an outage (AFTER). |
| iss-2610230716454430 | covered | Open: whether the build runner takes the drive's subscription-by-default route is owed to the product thinker; the idea presupposes the answer is yes. |
| iss-2610240932215918 | contradicted | Open, major: the claude runner the host-stepped build routes roles to runs bare on a key only, so no subscription route exists for a build role today; the drain drive's profile launch is the route that does run on a subscription. |
| adr-2610310235000654 | contradicted | Proposed: unattended planning (--auto-plan) runs its plan reviews on host agents only, so a terminal build cannot plan a draft; the idea's READY scope respects this, a wider reading would not. |
| itd-2611141610197753 | covered | Planned: makes a drive operable (carries on from one command, one status line); it and itd-2611251809580662 cover issue drives only, the operability a terminal build would inherit. The research note drive-operations-2026-11.md it cites records the three-day drain drive's watching session, six-attempt restart and 23-hour silent stall. This machine's private verdicts were searched too, and none covers or rejects the idea. |
| itd-2611251725209625 | covered | Draft, filed 2026-11-25: this idea, titled 'A build runs from a plain terminal, with no session open'; builds_on itd-2610222215485265 and itd-2611251725203320, refines itd-2609201916151817, and has no acceptance criteria yet. Its press release says 'as abcd drain already does', the wording the reframing below rejects, so planning it carries the reframing rather than a second intent. |
| dcn-2611251726008978 | covered | The product thinker's ruling of 2026-11-25: itd-2611251725209625 is the third of three parts and comes after the other two (itd-2611251725190796 and itd-2611251725203320), in the run that holds the drive's files. |
| itd-2611251725203320 | covered | Planned, with spc-2611251757236507 open: abcd start, the one command that makes the separate login and starts an attended session on abcd alone; abcd drain runs on the same login. It covers the separate login the idea names, not the unattended run (see the rejected alternative on abcd start). |

Corrected by hand on 2026-11-26, after this branch merged the default branch: the three rows above for itd-2611251725209625, dcn-2611251726008978 and itd-2611251725203320 were added, with the four correction searches at the foot of the table below; they reached the default branch after the grill was first run. The same correction restates the drain intent's remainder from spc-2611051628562376 in leg 1, in its row above, in both reviewers' lists and in the reframing; removes the private store's search row, whose figures this public record does not carry; and replaces a search of a local branch with searches of the merged head. Every other row stands as recorded.

### Searches

| Terms | Places | Count |
|---|---|---|
| build --drive | origin/main:.abcd/development/intents/ | 0 |
| build --drive | origin/main:.abcd/development/decisions/adrs/ | 0 |
| build --drive | origin/main:.abcd/work/decisions/ | 0 |
| build --drive | origin/main:.abcd/work/issues/ | 0 |
| build --drive | origin/main:.abcd/development/research/notes/ | 0 |
| build --drive | origin/main:.abcd/development/research/ideas/ | 0 |
| build --drive | origin/main:.abcd/development/brief/ | 0 |
| build --drive | origin/main:.abcd/development/specs/ | 0 |
| no-host drive | origin/main:.abcd/development/intents/ | 0 |
| no-host drive | origin/main:.abcd/development/decisions/adrs/ | 1 |
| no-host drive | origin/main:.abcd/work/decisions/ | 3 |
| no-host drive | origin/main:.abcd/work/issues/ | 0 |
| no-host drive | origin/main:.abcd/development/research/notes/ | 0 |
| no-host drive | origin/main:.abcd/development/research/ideas/ | 0 |
| no-host drive | origin/main:.abcd/development/brief/ | 0 |
| no-host drive | origin/main:.abcd/development/specs/ | 1 |
| plain terminal | origin/main:.abcd/development/intents/ | 15 |
| plain terminal | origin/main:.abcd/development/decisions/adrs/ | 2 |
| plain terminal | origin/main:.abcd/work/decisions/ | 3 |
| plain terminal | origin/main:.abcd/work/issues/ | 14 |
| plain terminal | origin/main:.abcd/development/research/notes/ | 3 |
| plain terminal | origin/main:.abcd/development/research/ideas/ | 0 |
| plain terminal | origin/main:.abcd/development/brief/ | 10 |
| plain terminal | origin/main:.abcd/development/specs/ | 5 |
| step 12 | origin/main:.abcd/development/intents/ | 1 |
| step 12 | origin/main:.abcd/development/decisions/adrs/ | 0 |
| step 12 | origin/main:.abcd/work/decisions/ | 7 |
| step 12 | origin/main:.abcd/work/issues/ | 7 |
| step 12 | origin/main:.abcd/development/research/notes/ | 0 |
| step 12 | origin/main:.abcd/development/research/ideas/ | 0 |
| step 12 | origin/main:.abcd/development/brief/ | 2 |
| step 12 | origin/main:.abcd/development/specs/ | 5 |
| abcd start | origin/main:.abcd/development/intents/ | 14 |
| abcd start | origin/main:.abcd/development/decisions/adrs/ | 6 |
| abcd start | origin/main:.abcd/work/decisions/ | 6 |
| abcd start | origin/main:.abcd/work/issues/ | 12 |
| abcd start | origin/main:.abcd/development/research/notes/ | 1 |
| abcd start | origin/main:.abcd/development/research/ideas/ | 0 |
| abcd start | origin/main:.abcd/development/brief/ | 5 |
| abcd start | origin/main:.abcd/development/specs/ | 7 |
| adopts the same four conditions / adopts it for abcd build | origin/main:.abcd/development/intents/ | 0 |
| adopts the same four conditions / adopts it for abcd build | origin/main:.abcd/development/decisions/adrs/ | 1 |
| adopts the same four conditions / adopts it for abcd build | origin/main:.abcd/work/decisions/ | 0 |
| adopts the same four conditions / adopts it for abcd build | origin/main:.abcd/work/issues/ | 0 |
| adopts the same four conditions / adopts it for abcd build | origin/main:.abcd/development/research/notes/ | 0 |
| adopts the same four conditions / adopts it for abcd build | origin/main:.abcd/development/research/ideas/ | 0 |
| adopts the same four conditions / adopts it for abcd build | origin/main:.abcd/development/brief/ | 0 |
| adopts the same four conditions / adopts it for abcd build | origin/main:.abcd/development/specs/ | 1 |
| intent's run; abcd build | origin/main:.abcd/development/intents/planned/itd-2611251809580662-a-drive-keeps-working-the-ledger-until.md | 0 |
| docfidelity gate callers; spec close | origin/main:internal/core/implement/loop/; origin/main:internal/surface/cli/ | 1 |
| build runs from a plain terminal | 3a4c6a64d:.abcd/development/intents/ | 1 |
| itd-2611251725209625 | 3a4c6a64d:.abcd/work/decisions/ | 1 |
| abcd start | 3a4c6a64d:.abcd/development/intents/ | 16 |
| abcd start | 3a4c6a64d:.abcd/development/specs/ | 9 |

## Leg 3 — Adversarial review

Conducted fresh-context and off-policy by an evaluator that did not carry
out the research and received the idea as an artefact of unknown
authorship — the evaluator-outside-the-loop principle applied to ideas.
Each reviewer's attempts, verdict and sources are kept under its own label.

Current state: ignored. The reviewers judged the idea as an ambition for
the product this repository builds: a gap between the idea and what the
product ships today was out of scope as a kill, and reads as what the idea
implies building.

### Reviewer reviewer-a — reframed

- **partial** — No real need: the only operating evidence (the drain drive's three-day run: a watching session, six restart attempts, a 23-hour silent stall) cuts against 'unattended', and an intent is a longer arc than an issue.
- **survived** — It breaks the host-delegated default.
- **partial** — Hand-back does not fit an intent: a drain hands one issue back and goes on, a build has one intent so a hand-back ends the run; the drive also hands back any lane touching the Makefile, scripts/ or hooks, which intents touch more often.
- **partial** — The wording 'entirely' and 'delivered': in human merge mode the merge waits for an approval and every shipped intent needs a saved docs review before the cut, so delivered can honestly mean only armed through the merge gate, or handed back.
- **partial** — The wording '(or a drive)' and 'separate login': a drive over several intents would choose what to build next, scheduling judgement abcd keeps with a person; 'separate login' understates the wall, cap, deadline and typed yes that matter.
- **partial** — Competitors already offer unattended terminal runs (codex exec, a hosted coding agent opening one pull request per task, claude -p) and steer automation onto API keys, weakening a subscription default.
- **survived** — Unattended building drops human judgement.
- **partial** — A separate build driver would duplicate the drain's walls, cap, outage handling and stop, and drift from it.

What the idea implies building:

- Ship itd-2610222215485265, including the three remaining steps of spc-2611051628562376: the live proof on the one walled login (MUST), the API-key route and the ruling on a lane paused past an outage.
- The build's runner takes the drive's subscription default (iss-2610230716454430); today it runs bare on a key only (iss-2610240932215918).
- The drive steps build lanes as well as drain lanes, with the build's stop semantics.
- A --drive flag or mode on build, wired into the CLI and the plugin command page.
- The key route's budget (itd-2611011230226453) and named profiles (itd-2611040344170206).
- Drive operability and long runs extended from issues to intents (itd-2611141610197753, itd-2611251809580662).
- A hand-back path when an intent's lane touches the Makefile, scripts/ or hooks.

Sources consulted: adr-2610222222326104, adr-2610310235000654, itd-2609201916151817, itd-2610222215485265, itd-2611011230226453, itd-2611040344170206, itd-2611141610197753, itd-2611251809580662, spc-2609202134338445, iss-2610222000271964, iss-2610230716454430, iss-2610240932215918, dcn-2610230716408975, dcn-2611031722512037, https://code.claude.com/docs/en/headless, https://code.claude.com/docs/en/legal-and-compliance, https://learn.chatgpt.com/docs/non-interactive-mode, https://docs.github.com/en/copilot/concepts/agents/coding-agent/about-coding-agent.

### Reviewer reviewer-b — reframed

- **partial** — Multi-step specs make 'unattended to delivered' incoherent: an intent's spec closes only on the closing lane after every earlier lane merged, and in human merge mode each lane's pull request waits for an approval, so delivery unattended holds only in auto mode.
- **survived** — The fidelity audit and spec close cannot run unattended.
- **partial** — Size of change: intents touch the Makefile, scripts/ and hooks far more than issue fixes, and a drive hands those lanes back, so many intents can never be delivered unattended in this repository.
- **survived** — Parallel lanes make an unattended intent run unsafe.
- **partial** — Subscription terms: the route sits in the end-user carve-out, but limits assume ordinary individual usage and an intent run multiplies agents; --bare is to become the default for -p and the vendor may enforce without notice.
- **partial** — Operating it: the drain's three-day run needed a watching session, stalled 23 hours silently and took six restart attempts; the operability intents cover issues only.
- **survived** — Competitors' hosted agents already take a task to a pull request.
- **survived** — Incoherent with the product's commitments: it is the planned widening of the host-delegated boundary, and the READY scope matches unattended planning staying on host agents.

What the idea implies building:

- itd-2610222215485265 shipped, with the three remaining steps of spc-2611051628562376 (the live proof on the one walled login, a MUST; the key route, waiting on itd-2611011230226453 and itd-2611040344170206; the paused-lane ruling).
- Spec step 12 itself, with AGENTS.md's boundary widened to name it.
- A ruling on iss-2610230716454430 and a fix for iss-2610240932215918, or the build using the drive's own profile route.
- The drive's set-aside receipts and stuck count extended to intent runs.
- Unattended timing for the docs review the close and the release cut need.
- Stacked dependent lanes (itd-2610230856403803), or accepting merge-paced runs.
- Operability and long-running-drive work extended from issues to builds.

Sources consulted: https://code.claude.com/docs/en/legal-and-compliance, https://code.claude.com/docs/en/headless, https://code.claude.com/docs/en/env-vars, spc-2609202134338445, adr-2610222222326104, itd-2609201916151817, iss-2610240932215918, itd-2610230856403803, itd-2610222215485265, adr-2610310235000654, iss-2610230716454430, itd-2611141610197753, itd-2611251809580662.

The reviewers agree, so no ruling was needed.

## Rejected alternatives

- **A separate build driver written beside the drain drive** — it would duplicate the walls, cap, outage handling and stop semantics and drift from them; the build steps through the drive's one engine
- **A drive that works through several intents and chooses which to build next** — choosing the next intent is scheduling judgement abcd keeps with a person; one typed command covers one READY intent
- **Extending the terminal build to draft intents through --auto-plan** — adr-2610310235000654 keeps the plan reviews on host agents and acceptance criteria are a person's; nothing unattended writes them
- **Routing the build's roles to the existing claude runner with an API key** — it runs bare on a key only (iss-2610240932215918), so a subscription-only person has no route; the drive's walled profile launch is the route that runs on a subscription
- **Use abcd start (itd-2611251725203320, an attended session on the separate login) and run /abcd:build in it** — that is a host session the person keeps open, so it is not unattended; it stays the attended alternative, not this idea

## The reframing

Not a new idea: it is spc-2609202134338445 step 12, the build's no-host drive under adr-2610222222326104's four conditions, and the surviving shape is that step worded honestly. Typed by the person at a plain terminal, `abcd build <itd-N> --drive` runs the drain drive's own engine, walls and profile login over one READY intent's lanes, never over a draft: it never plans, never writes acceptance criteria and never picks the next intent. It ends in one of two named outcomes: every lane landed and the spec closed where the merge gate's facilitator.mode lets it, or each lane armed and waiting for approval or handed back with its reason (a lane touching the Makefile, scripts/ or hooks among them); 'delivered, or a named hand-back', not 'delivered'. It is not 'as drain already does': an intent adds several merge-ordered lanes, the fidelity audit and the spec close, and the drive's stop semantics must be extended to intent runs. It waits for the drain drive to prove it stays running and says when it stalls (itd-2611141610197753). Whether it may wait only on the drain intent's subscription-route steps (the built drive, and the live proof on the one walled login that spc-2611051628562376 holds open as its first, MUST step) rather than on the whole remainder, whose key route a subscription-only person does not use, is the product thinker's to rule; until then step 12 waits on itd-2610222215485265 shipping. It is the idea the draft itd-2611251725209625 already holds, which dcn-2611251726008978 schedules after itd-2611251725190796 and itd-2611251725203320.

## What follows

The idea as posed does not survive, but the reframing recorded above
does. Any graduation to a draft intent carries the reframing, not the
original wording.
