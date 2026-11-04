---
id: itd-2611041549483696
slug: once-a-day-an-unattended-run-writes-one
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2611031342011261, itd-2610240850336695, itd-2610032150577708]
related_issues: [iss-2610130501353955]
severity: major
origin: researcher-authored
production_mode: hand-written
---

# An unattended run writes one private daily digest page

## Press Release

> **Once a day, an unattended run writes one dated page that says what merged, what stalled, what each key spent, and what waits for the person; it sends nothing else.** Iris opens the page when they choose, from the shared folder they picked at setup on their own computer. It reads top to bottom: the pull requests that merged since the last page, each with the record it resolved or delivered; the lanes that stalled, each with the reason the run recorded and what it left; the spend of each named key against its cap, with how many stages the key carried and why the subscription did not; the releases cut and the refusals that stopped one; and last, the list of what waits for Iris alone, drafts held on a choice, questions queued, a stop the run honoured. No alert, no message, no notification arrives: the page is where the run reports. "A run that pings me is a run I cannot leave alone," said Iris, a product thinker. "One page a day, in one place I choose to look, is the whole arrangement."

## Why This Matters

The product thinker ruled that an unattended run keeps the person informed through one private daily digest page carrying what merged, what stalled, spend per key, and what waits for them, and through nothing else (dcn-2611031615094092). The ruling names the page; nothing in abcd writes it. On 2026-11-04 a coordinating session wrote the page by hand, revised more than once, beside a hand-kept spend log per key and a hand-kept queue log; the launch-ledger draft records that on 2026-11-01 three sessions each tallied their own spend by script and had to ask one another what was left of the key (itd-2611031342011261). The hand-written page took the coordinator's time from the work and could be wrong in the ways a session's memory is: it reported a release under a version the binary had not derived.

Every figure the page needs is, or is about to be, a record fact. Merged pull requests and their `Resolves:` and `Delivers:` trailers are the forge's and the ledger's; a stalled lane is a run state's paused await, hand-back or outage, and the log's `stop`, `refusal` and `fallback` events (`abcd implement log`); spend per key is the launch ledger's per-key sum (itd-2611031342011261, decisions 1 and 11), which also answers the run-start budget check that never compares anything (iss-2610100913096346); what waits for the person is the autonomous mode's queue (`abcd mode queue`), the drafts held because an open question has more than one defensible answer (dcn-2611031615094092), and the refusals a release cut waits on. The grant draft (itd-2610240850336695) promises a list of each push and pull request a session made under a grant, and the dashboard (itd-2610032150577708) promises what waits on the person first; this page is the daily cut of both, composed from the same records, so neither is duplicated.

The place was the hard part, and the product thinker settled it with the stop switch's (dcn-2611041550139264): the page is a dated file in the shared folder chosen as an option at setup, on a machine no other person shares, the one file family outside abcd's owner-only rule beside the stop; a page served over a private network comes later. iss-2610121528179959 names the same gap for handovers, and the choice answers it for this page.

## Mechanism

We expect the page to be right where the hand-written one was wrong because every line on it is derived from a record (the ledger, the run states, the log, the forge's merged pull requests, the mode's queue) by the first abcd move after the day's boundary, with nothing typed by an agent; a fact abcd cannot source is absent, not estimated. The claim is shown wrong if a page carries a figure no record holds, or if two pages are written for one day.

## Scope Conditions

- Holds for one machine's runs, composed by whichever abcd process in the account that runs the work first passes the day's boundary, that account's ledger being the one that holds the launches; a move in another account composes nothing. A hosted run reports through its own summary.
- Holds where the person chose the shared folder at setup, on a machine no other person shares (dcn-2611041550139264); without it no page is written and `abcd ahoy` says so.
- The page is derived: every line names the record, run, ledger entry or pull request it rests on, and nothing on it is typed by an agent. A fact abcd cannot source is absent, not estimated.
- Spend figures are the harness's estimates at its price table, as the ledger keeps them; the page says so, and a subscription stage shows no money.
- Private means reachable by the person and no one else: no forge, no public branch, no mail; key service names and account names never appear on it.
- One page a day, dated, kept beside its predecessors; the page is never pushed to the person by any channel.

## Acceptance Criteria

- **Given** the first abcd move after the day's boundary, **when** it runs, **then** one page for the previous day is composed from that account's records and written to the shared folder; a later move the same day in that account rewrites the page whole from the records, never appends to it; a move in any other account writes nothing; a day with no run writes a page that says so; and a stop (itd-2611041549471712) composes that day's page itself before the last run it halted ends, in the run account, so a stopped day is never without one.
- **Given** the day's merges, **when** the page lists them, **then** each shows the pull request, the records its trailers resolved or delivered, and the release it reached if one was cut, read from the forge and the ledger, never from a session's notes.
- **Given** a lane that stalled, **when** the page lists it, **then** it shows the run, the lane, the stage, the recorded reason (paused, handed back, outage, rate limit, stop) and what the lane left (a receipt uncollected, work saved aside, a pull request waiting), with the command that picks it up.
- **Given** the named keys, **when** the page shows spend, **then** each key shows the month's spend against its cap, the day's spend, the stages it carried, and for each the window reading that sent the stage to the key; a key at its cap is marked, and a subscription stage is counted, never priced.
- **Given** what waits for the person, **when** the page ends, **then** it lists each held draft with the question that held it, each queued act, each release refusal, and any stop in force (itd-2611041549471712), each with its record id; an empty list says so.
- **Given** the page exists, **when** any abcd surface would alert the person (a push, a message, a notification hook), **then** it does not; the page's existence is the whole report, and a test proves no outbound channel is touched.
- **Given** the page's place, **when** it is written, **then** it is readable by the person's own account, and nothing in it identifies a key by its service name or an account by its name.

## Open Questions

_None open. The question of place is decision 1 below; the two the interview did not reach are settled on the draft's proposal in decisions 2 and 3, each saying why no question was put._

## Decisions

1. 2026-11-04, the product thinker (dcn-2611041550139264): the page is a dated file in the same shared folder as the stop switch, on a machine no other person shares; the scratch draft asked that one choice hold for both, and an unshared machine answers the privacy cost, so this followed from the stop's ruling and no question of its own was put. A page served over a private network is later work.
2. **Who composes it**: any abcd move in the account that runs the work that first crosses the day's boundary, so a page appears whether or not a coordinator is alive, and a later move in that account may rewrite the day's page; a move in the person's own account, whose ledger holds none of the launches, composes nothing. No question was put because the alternative, the coordinating session alone at a fixed hour, loses the page whenever the coordinator is what died, and a page that says the day was quiet is the signal that the run is alive; settled on the draft's proposal, with the standing stance that abcd ships a sensible default the person configures (dcn-2611041550129278) leaving the hour configurable.
3. **A day with nothing to report** writes a page that says so, proving the run is alive. No question was put because the first criterion already needs a page per day for the stop and the queue to be reported; settled on the draft's proposal.

## Decomposition

Hand-run four-piece table, 2026-11-04; the product thinker confirmed the routing (dcn-2611041550139264).

| Part | Kind | Home |
|---|---|---|
| the daily page, its five sections, composed from records once a day, written to the shared folder | capability | this intent |
| the run reports through a page the person opens, never through a channel that reaches them | trust rule | one line in the ADR owed with the stop switch (itd-2611041549471712), extending adr-2610240850193281; dcn-2611031615094092 is its ground |
| a report is derived from records or it is not written | stance | already abcd's (the changelog is derived, the site cannot drift: `commands/site.md`); no new principle |
| the file's shape, the day's boundary, the reader of each section | plumbing | the brief's loop and dashboard chapters, through this intent's spec |

Verdict: FILE-AS-IS, confirmed. Typed links: `builds_on` itd-2611031342011261 (spend per key), itd-2610240850336695 (the grant's daily list becomes a section), itd-2610032150577708 (the dashboard's "what waits on me" is the same list, served live). Flagged, never auto-classified: the dashboard spec's rule that every page comes from the record site's one renderer would make the digest a site page; the ruling on place keeps it a file, and the renderer may still compose it.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
