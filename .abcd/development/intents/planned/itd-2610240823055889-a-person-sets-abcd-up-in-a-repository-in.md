---
id: itd-2610240823055889
slug: a-person-sets-abcd-up-in-a-repository-in
spec_id: spc-2610241007434505
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610030810370060]
severity: major
origin: researcher-authored
production_mode: hand-written
impact: breaking
---

# Setting abcd up is short, plain, and finishes

## Press Release

> A person sets abcd up in a repository in one short sitting, and it finishes. Setup asks only the questions whose answers it cannot work out for itself, one screen at a time in a polished terminal interface, each with a sentence or two rather than a page; it does what each answer allows at once, shows plainly what it changed, and ends either with abcd fully working (its command on the PATH, the status line wired, the commit identity right) or with the one next step that is still needed and why. Nothing is left half-done behind a generic yes or no, and running it again from a source checkout works the same way as from a release.

## Why This Matters

_Drafted by the agent from the 2026-10-24 setup session; not yet confirmed by the product thinker._

On 2026-10-24 the product thinker set abcd up in abcd's own checkout and found it "complicated, too much text, and doesn't seem to be working". Getting to a clean install took five runs of `ahoy install` (one refused outright, one with every question declined, three at a terminal), a hand-typed `git config`, and the README's install line pasted by hand. What the session showed:

- **The status line only worked after a step nobody was told about.** The status-line offer was asked and answered through all its element switches, then refused afterwards because no PATH entry was on record (iss-2610240823574047). The only way on was the README's install line: one very long command, pasted into a terminal by hand.
- **After setup, abcd does not recognise the command it just installed.** With `~/.local/bin/abcd` on PATH and recorded in `~/.abcd.noindex/path-entry`, the detection pass still reports an empty `install_mode`, beside the standing note that the plugin folder was not found (iss-2610240916382802).
- **The identity fix was hidden behind a generic yes or no.** A no to "Change the settings listed above?" silently skipped the identity question the person wanted answered (iss-2610240824005655).
- **The committed identity pin names one person**, so setup and the pre-commit hook tell any other contributor to commit under the maintainer's name (iss-2610240830397793).
- **A run from the source checkout is always refused as stale**, because `go run` never stamps a build, although the repository's own rules make `go run` the only sanctioned way to run abcd there (iss-2610240824047504).
- **Tool installs assume admin rights.** Homebrew on this machine belongs to another account, so `brew install gitleaks` failed on every run, and the failure note passed on Homebrew's `sudo` advice (iss-2610240824571059). A by-hand Go build of the tool into `~/.local/bin` worked, which shows a route without admin rights exists; it is evidence, not the route chosen.
- **Too much text, too many questions.** Each run asked up to thirteen questions, several of them only "go on to the offer listed above?", and a clean run's report still ran to a screen of detail and notes.

## Mechanism

- We expect setup to finish in one run because it checks everything before asking anything, and asks a question only once the steps it depends on can run.

## Scope Conditions

- macOS and Linux. <!-- cond: cond-2610241007438041 -->
- An account with or without admin rights. <!-- cond: cond-2610241007432893 -->
- An interactive terminal; anywhere else, setup is answered by options or an answers file. <!-- cond: cond-2610241007434646 -->
- A released abcd and a source checkout of abcd. <!-- cond: cond-2610241007439022 -->

## Acceptance Criteria

_Each criterion confirmed by the product thinker, one at a time, on 2026-10-24._

- Given no git, no repository or no agent harness, when setup runs, then it stops before any question with one message naming what is missing.
- Given setup can run, when it starts at a terminal, then it checks everything first, shows one plan screen listing every change and every default with where to change it, and asks once: proceed, customise or cancel.
- Given a step only a person can approve (the commit identity, a download, the assistant's settings, the shell settings), when setup reaches it, then it is its own question naming the value and where it is written, whatever the answer to proceed.
- Given abcd is not on the PATH, when setup runs, then it offers the abcd download before the status line, and on a yes wires the status line in the same run; it never asks about the status line when it cannot wire it.
- Given an account that has never used abcd and the suggested answers, when setup runs at a terminal, then it asks at most four questions (the abcd download, the shell settings, the plan screen, the commit identity), plus one question for each missing tool it offers to install.
- Given any setup question, when it is shown, then it is one line, and answering `?` shows the full explanation and asks again.
- Given setup ends, when its report is printed, then it has three parts: done (one line per change, with where), needs you (a numbered checklist, each with the exact step and why), and next (one step).
- Given a repository that is fully set up, when setup runs again, then it prints one line and changes nothing.

## Open Questions

- resolved: Should setup work on an account without admin rights, installing every tool it offers for that account alone? (raised 2026-10-24; see Decisions)
- resolved: Should setup work for more than one person committing to the same repository, so that no contributor is ever asked to take on another's identity? (raised 2026-10-24; see Decisions)
- resolved: Should setup put abcd's own command on the PATH itself, so that the status line never depends on the README's install line? (raised 2026-10-24; see Decisions)
- resolved: When a tool setup needs is already installed some other way (by Homebrew, a package manager, or by hand), what does setup do? (raised 2026-10-24; see Decisions)
- resolved: Which basics does setup check before anything else, and how does it report a missing one? (raised 2026-10-24; see Decisions)
- resolved: Do the research proposals under Research stand: one plan and one proceed, two to four questions on a first run, one line per question with `?` for more, a one-line clean re-run? Two of them sit against brief invariant 1 ("every prompt shows current state, consequences of each option, and how to change later. No silent defaults."), so accepting them amends that invariant. (raised 2026-10-24 by the design and record reviews) (answered 2026-10-24; see Decisions)
- resolved: The basics decisions disagree with each other and with the research: `curl` and a checksum tool are needed only by the README's install line, `gh` is not needed for a verified install, and without git, a repository or a harness there is nothing to set up, so those stop setup rather than wait for the end. Which basics block, and which go on the end-of-run list? (raised 2026-10-24 by the design review) (answered 2026-10-24; see Decisions)
- resolved: Is this one intent, or a setup-flow intent plus separate records for the commit identity, third-party tool installs and shell settings, with the stale `go run` refusal and the status-line order fixed as issues on their own? (raised 2026-10-24 by both reviews) Split by the product thinker on 2026-10-24; see Decisions.

## Research

State of the art (2026-10-24; sources: clig.dev, rustup, uv, Homebrew's installer and documentation, the GitHub CLI, mise, aqua, flutter doctor, create-next-app, Nielsen Norman Group on progressive disclosure, and Sampath, Merrick and Macvean, "Accessibility of Command Line Interfaces", CHI 2021). Proposals for the planning interview, not yet confirmed:

- Detect everything first, show one short plan grouped by where each change lands, and ask once: proceed, customise, or cancel (rustup, Homebrew's installer, Terraform, create-next-app).
- A question only a person can answer (the identity, installing a tool, the harness's settings) is asked as itself, naming the value and where it is written, never behind a generic approval.
- Steps run in dependency order, and a step that cannot run is never asked about.
- A first run asks two to four questions; everything with a reasonable default is decided and listed in the plan with where to change it.
- Each question is one line; answering `?` shows the full explanation.
- The report ends in three short parts: done, needs you (a numbered checklist), next.
- The prerequisite pass reads like a doctor: ok, needs you, or missing, each with one fix line, and blockers kept apart from notes.
- Tools install from a per-tool table checked in-process: GitHub's asset digest, the release's checksums, and the signed build record where the project publishes one (gh does; gitleaks is checksum-checked only); `gh` is not needed for a verified install.
- abcd keeps its own question package rather than adding a terminal UI framework, and makes it calmer: no animation, `NO_COLOR`, the numbered mode always available.
- A clean re-run prints one line.

## Decisions

- 2026-10-24, the product thinker: before asking anything, setup checks everything abcd needs to work, and the list grows as abcd does; today that is git (present and recent enough), the agent harnesses it wires into and whether each reads `AGENTS.md`, `~/.local/bin` existing and on the PATH, the download tools it relies on (`curl`, a checksum tool), `gh` where signed build records are checked, and, in a source checkout of abcd, Go at the version `go.mod` declares.
- 2026-10-24, the product thinker: a missing basic does not stop setup. At the end, setup offers to install what is missing, or, where it cannot or the person declines, gives a checklist in plain language of what to do by hand.
- 2026-10-24, the product thinker: setup puts abcd's own command on the PATH itself, in the same run, after one question that names the download; the status line never depends on the README's install line (iss-2610240823574047).
- 2026-10-24, the product thinker: this plan is split. It keeps the setup flow (what is checked first, what is asked and in which order, and the report). Three parts are their own drafts: every contributor commits as themselves (itd-2610240918547728), abcd installs the tools it needs for the person alone (itd-2610240918594138), and setup offers to put abcd's folder on the PATH (itd-2610240919030303). The stale refusal of a source-checkout run and the status-line order are fixed as issues on their own (iss-2610240824047504, iss-2610240823574047).
- 2026-10-24, the product thinker: a missing git, a folder that is not a repository, or no agent harness stops setup at once, before any question, with a plain message saying what is missing; every other missing basic goes on the end-of-run list. The earlier decision that a missing basic does not stop setup holds for those others only.
- 2026-10-24, the product thinker: setup's questions are short (one line, with `?` showing the full explanation), and defaults are decided for the person and listed on the plan screen; what each answer does is shown in full only on `?`. This changes brief invariant 1 ("every prompt shows current state, consequences of each option, and how to change later. No silent defaults.") for setup, and the change is owed an ADR that amends the invariant before the work ships.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: we expect a first-time setup to end working in one run; it is shown wrong if a fresh account still needs a hand-run step or a second run.
