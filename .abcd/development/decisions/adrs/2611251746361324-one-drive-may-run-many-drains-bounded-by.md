---
id: adr-2611251746361324
slug: one-drive-may-run-many-drains-bounded-by
status: proposed
date: 2026-11-25
supersedes: null
superseded_by: null
refines: [adr-2610222222326104]
related_intents: [itd-2610222215485265, itd-2611141610197753, itd-2611251809580662]
related_rfcs: []
related_adrs: [adr-2610222222326104]
---

# ADR-2611251746361324: One drive may run many drains, bounded by its end time and an optional cap, and never extends its own bound

Typed links: `refines` [adr-2610222222326104](2610222222326104-abcd-drives-agents-with-no-host-session.md)

## Context

[adr-2610222222326104](2610222222326104-abcd-drives-agents-with-no-host-session.md) lets abcd drive agents with no host session on four conditions. Its third, "bounded", has a drive end at its cap, at an empty queue or on the person's signal, and its amendment of 2026-11-05 lets the person name an end time with `--until`. A drive today is one drain: the drain's cap (eight unless `--max` says otherwise) is fixed when the drain begins, and a drive ends when the drain stops at that cap or finds nothing eligible.

The goal of the planning of 2026-11-25 is drives on several machines and accounts that keep working one repository's ledger for as long as the person wants, with nobody watching. Two of the drive's ends are passing conditions on a shared ledger. A cap of eight ends a drive a few hours in, and an empty ledger fills again when another machine captures an issue, a person rewrites a handed-back record or a hold lapses. By then the drive is gone, and a person must start it again.

The product thinker ruled on 2026-11-25 that a drive no longer ends at a drain's cap: it begins the next drain itself. It ends at its end time, which defaults to seven days from its start and which a per-repository default may change, at an optional lane cap if the person gives one, or on a stop. No daily lane count was asked for. What bounds a day is the per-drain limit on lanes in work, one by default ([dcn-2611251747513582](../../../work/decisions/dcn-2611251747513582-the-product-thinker-ruled-on-the.md) records it with the per-person ceiling), and the person's subscription.

A yes that covers many drains carries more weight than a yes that covers eight lanes, so what may move the bound after the yes is the trust question this record settles.

## Decision

We will let one drive run many drains, one after another, under one yes and one bound.

1. **The bound is an end time and, optionally, a cap.** A drive ends at its end time (`--until`), at its lane cap when the person gives one (`--max`), or on the person's stop. It does not end when a drain reaches its cap or finds nothing eligible: it begins the next drain in the same process, keeping the flags the person gave at its start (the pace, the sub-agents, the fix rounds and the in-work limit), the ended drain's skip list as its state left it, and its profile, and waits through an empty ledger by the drain's existing look-again interval. A setting the person did not give is read again from the layers for each drain. A skip, given at the start or added or lifted mid-drive, lasts for the life of the drive and clears when the drive ends, so an empty look that ends a drain never undoes it with nobody watching (dcn-2611252011565595, dcn-2611252026516969).
2. **Every drive has an end time.** With no `--until`, the end time is seven days from the start, or the repository's default when it sets one. The start's `ends` line names the end time, with the cap where one was given, before the yes; the per-drain limit on lanes in work and the command that resumes the drive are on `?` and in the full plan the drive writes to its log, and the command is printed once after the yes. The yes covers that one drive, and it is never remembered.
3. **A drive never extends its own bound.** Nothing the drive does moves its end time later or raises its cap: not beginning a new drain, and not re-executing itself for a new release, which carries the end time and the lanes opened so far and reads no default again. The command a drive prints for carrying on after a reboot, a logout or a crash names the same end time, so carrying on does not move the end; only a start the person types with a later `--until` sets a later one.
4. **No repository file extends a bound.** A repository's default end time (`drive.until_days`, 1 to 31) is read from the commit origin's default branch stood at when the checkout last fetched it, never from a working tree, a lane's worktree or any other branch, since a lane's agent can write those and cannot write the default branch. The same holds for the repository's other drive numbers, the person's reservation ceiling and the drain's in-work limit. The default end is read once, before the yes, and is shown in it; it proposes the bound the person agrees to and never moves one already agreed. A change to the repository's configuration while a drive runs, whether made by a peer's landing, a fast-forward or the drive's own lanes, moves neither its end time nor its cap. This applies condition 1 of adr-2610222222326104 (a repository's configuration can never start or widen a drive) to a drive that spans many drains.
5. **Every other condition of adr-2610222222326104 stands.** The person's own invocation, the walled account and route, every wait having a deadline no later than the drive's end, and what the drive cannot finish going back.

## Alternatives Considered

- **One drive, many drains, under an end time and an optional cap (chosen).** It keeps a drive working through the passing ends of a shared ledger, and it keeps the bound the person's: the yes names it, and nothing after the yes moves it.
- **Keep one drain per drive, with a larger default cap.** Rejected: an empty ledger still ends the drive, and any cap is a number of lanes the person must guess against a ledger that other machines change.
- **A daily lane number as the rate.** Rejected: the product thinker asked for none. The per-drain limit on lanes in work and the subscription already bound what one drive does in a day, and a second figure would be one more to explain.
- **An unbounded drive that runs until stopped.** Rejected: condition 3 of adr-2610222222326104 holds so that a drive cannot run on unseen, and a drive whose bound is a stop alone runs on after a person forgets it.
- **A repository setting that may lengthen a running drive.** Rejected: it would let a change a person did not agree to, landed by anyone who can write to the repository, keep agents running on the person's subscription past the time they named.

## Consequences

- **Easier:** a person starts a drive once and it works for a week, through empty spells and drains that end, on every machine where they start one.
- **Harder:** a drive process stays alive, mostly idle at times, until its end time, and a person who wants it to end sooner stops it or names an earlier `--until`. The estimate of agent runs the yes shows can no longer be read off a cap of eight; the yes shows the end time and the cap where one is given, and a drive on a forgotten machine may hold reservations and spend for up to its seven days.
- **Condition 3 of adr-2610222222326104 is refined, not reversed:** "it ends at the cap, at an empty queue, or on the person's signal" now reads "it ends at its end time, at its cap if one was given, or on the person's stop". The amendment of 2026-11-05 kept eight hours on each wait for a drive with no `--until`; with a default end time every drive has a bound, so no wait of eight hours, outage or limit ends a drive any more: it waits until its end time (itd-2611251809580662, decision 1).
- **New obligations:** the drive keeps the finished drain's state beside the current one, so a postmortem keeps the counts. The brief's drain chapter (`.abcd/development/brief/04-surfaces/35-drain.md`) gains the invariant that no drive and no repository file extends a drive's bound, written in the change that builds it.
- **Restart after a reboot is not part of this.** The drive's full plan says that a reboot, a logout or a crash ends it, and the drive prints the one command that resumes it after the yes; nothing that starts by itself is installed (itd-2611041656024409).
