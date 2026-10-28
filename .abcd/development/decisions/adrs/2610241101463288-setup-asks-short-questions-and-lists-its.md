---
id: adr-2610241101463288
slug: setup-asks-short-questions-and-lists-its
status: accepted
date: 2026-10-24
supersedes: null
superseded_by: null
related_intents: [itd-2610240823055889]
related_rfcs: []
related_adrs: [adr-38]
---

# ADR-2610241101463288: Setup asks short questions and lists its defaults on one plan

## Context

Brief invariant 1 says every prompt shows the current state, the
consequence of each option and how to change the answer later, and that
there are no silent defaults. Applied to `abcd ahoy install`, it produced the
setup the product thinker ran on 2026-10-24: up to thirteen questions a run,
each a page of explanation, several of them only "go on to the offer listed
above?", and a report that ran to a screen of detail. Setup took five runs to
finish (itd-2610240823055889, Why This Matters).

The research for that intent (clig.dev, rustup, Homebrew's installer,
create-next-app, Nielsen Norman Group on progressive disclosure, and the
CHI 2021 study of command-line accessibility) found the same shape in the
installers that finish in one run: check everything first, show one plan
grouped by where each change lands, ask once, and keep every question to one
line with the full explanation one keystroke away. The product thinker
decided on 2026-10-24 that setup's questions are short (one line, with `?`
showing the full explanation) and that defaults are decided for the person
and listed on the plan screen, and recorded that the change is owed an ADR
amending the invariant before the work ships.

## Decision

We will hold setup to an amended invariant 1:

- Setup checks everything before its first question and shows one plan
  screen listing every change it would make and every default it would
  take, each default with where to change it. It asks once: proceed,
  customise or cancel.
- A default on the plan screen is not a silent default: it is listed, with
  its value and where to change it, before anything is written.
- Every setup question is one line. Answering `?` shows the whole question
  as invariant 1 describes it, the current state, what each answer does and
  how to change it later, and asks again. The record of every answer keeps
  the whole question.
- A step only the person can approve (the commit identity, a download, the
  assistant's settings, the shell settings) is its own question, naming the
  value and where it is written, whatever the answer to the plan.
- Every other abcd question keeps invariant 1 as it stands.

The download the plan lists is fetched only on a yes to the question that
names it: that question is the fetch's documented meaning, as adr-38 asks of
every fetch.

## Alternatives Considered

- **Keep invariant 1 whole for setup.** Rejected: the 2026-10-24 session
  showed it produced a setup that did not finish, and the explanation that
  was meant to inform drowned the questions that mattered.
- **Short questions with no `?`, the explanation in the docs only.**
  Rejected: the person answering would have to leave the question to learn
  what an answer does, which is the silent default the invariant forbids in
  another form.
- **One plan screen and short questions, with `?` for the full question.**
  Chosen: the plan shows every change and default before anything is
  written, so nothing is silent, and the full explanation stays one
  keystroke away at every question.
- **Ask nothing and take every default, as `--yes` does.** Rejected: the
  commit identity, a download and the assistant's settings are a person's to
  approve, and `--yes` already refuses them for that reason.

## Consequences

- A first run at a terminal asks the plan screen and the person-only
  questions only: at most four, the abcd download, the shell settings, the
  plan and the commit identity, plus one for each missing tool it offers to
  install, and one for the status line where it can be wired.
- The approve.<category> ids of the answers file give way to one
  `approve.plan` answer; a file naming an old id is refused, naming the new
  one, so the change is breaking.
- The asking-rules check in `abcd guard hook` is unchanged: it judges the
  questions an agent puts through the host's question tool, and the plugin
  page still relays setup's questions with their whole material.
- Every setup question needs a one-line form that names its value and where
  it is written; core composes it (`ahoy.BriefQuestion`), so no front door
  invents its own words.
