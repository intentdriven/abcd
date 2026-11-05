---
id: adr-2611050740453233
slug: a-stop-is-recorded-state-a-person-sets
status: proposed
date: 2026-11-05
supersedes: null
superseded_by: null
related_intents: [itd-2611041549471712, itd-2611041549461012]
related_rfcs: []
related_adrs: [adr-2610240850193281, adr-2610181003541997]
---

# ADR-2611050740453233: A stop is recorded state a person sets, and one shared folder is the exception to the owner-only store

## Context

A person cannot interrupt a tool call as it runs, so watching a run is no control (dcn-2611031722512037). The same fact sets where a stop can act: between one move of a run and the next, never inside a call. A signal reaches only the terminal it is sent to, and the person is usually not at that terminal.

The person's seat and an unattended run's seat may be two accounts on one machine. Every store abcd has admits only a file the reading account owns and no other account can write. That rule is what makes those stores trustworthy, and it also means none of them can carry a word from one account to another. The autonomous mode's store has this shape: a grant permits work (adr-2610240850193281) and its store is read before a stop (adr-2610181003541997), but neither withdraws anything from an account other than its own.

The product thinker ruled that the stop lives on the machine that runs the work, in a shared folder chosen as an option at setup, on a machine no other person shares; that the store ownership rule admits this one exception; that an agent in a call finishes that call; and that a stop releases the run's claims (dcn-2611041550139264). A signal in a terminal run is a stop too, and a second press kills at once (dcn-2611041608057544).

## Decision

We will make a stop recorded state that a person sets, and we will admit one shared folder as the exception to the owner-only rule.

**A stop is recorded state.** The person's own command writes a stop file, and every run reads it before each move: no lane opens, no stage is handed out and no agent starts. An agent in a call finishes it and its receipt is collected. With no agent left in a lane, the lane's uncommitted work is saved aside. Only then are the run's claims released and the run marked halted. A save that fails holds the lane and its claims.

**The person's command sets and lifts it.** Setting a stop, lifting it and choosing the folder are the person's acts. The guard refuses each to an agent under any program name, refuses a write to the stop file or the setting through the shell or the host's file tools, and the verbs refuse a caller with no terminal or under a host session. An agent's part in a stop is to finish its call, hand back its receipt and end. A lift is recorded. A run that let its claims go stays halted after the lift until a take-over reserves the work again.

**One folder is the exception to the owner-only rule.** The reason is the two seats: the stop has to cross from one account to another on one machine, and a local file is the one carrier that needs no network and no service. The exception is as narrow as that reason:

- The folder is chosen at setup, in each account that takes part, and declining leaves every other part of abcd working. Without it a stop lives in the account's own home and reaches only that account's runs.
- The folder is a real directory owned by root or the person, and so is each parent above it; one others can write carries the sticky bit. The stop sits in a directory of the person's own inside it.
- The file is a request to stop and nothing more. It is judged on one descriptor: a regular file, within a small bound, owned by the account named at setup. No byte of it is read, and who set it and when come from the descriptor.
- Every other store keeps the owner-only rule.

**A run holds to what it started under.** A run fixes the folder and the person at its start and reads that folder before each move. A setting that has gone or changed during the run, or a home that cannot be found, holds the run with nothing let go. Only a read that saw clearly and found no stop counts as a lift.

## Alternatives Considered

- **A signal to the run's process.** Rejected as the way to stop a run from elsewhere: it reaches one terminal or needs the right to signal another account's process, and it leaves no record a run can read at its next move. A signal stays as the terminal run's own way to take the same stop.
- **Carrying the stop in the autonomous mode's store.** Rejected: that store admits only files the reading account owns, which is why it cannot cross accounts, and a grant permits where a stop withdraws.
- **A stop over a private network or through a cloud adapter.** Not built now, by the ruling. It is later work, and a machine that needs it has the local stop in the meantime.
- **A folder chosen at setup, read as a recorded state. Chosen.** It needs no service, it works when the network does not, and its exception to the owner-only rule is one file.

## Consequences

- Every move of every run reads the stop, which is one open and one status call on a small file.
- A stop takes hold at the next move, so a long call still runs to its end, and a process killed outright is treated by the restart path as having lost its agents.
- A run that took a stop needs a take-over before it steps again, so nothing resumes by itself when a stop lifts.
- A person who moves the shared folder lifts the stop first, since the choice does not move while a stop stands in it.
- Setup gains one option, and the folder's rule is checked on every read.
- A brief invariant states this rule once this record is accepted, which is the person's act.
