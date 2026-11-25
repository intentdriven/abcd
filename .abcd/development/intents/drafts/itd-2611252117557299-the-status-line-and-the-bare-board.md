---
id: itd-2611252117557299
slug: the-status-line-and-the-bare-board
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610150602547271]
severity: critical
refines: [iss-2610262036211949]
impact: additive
related_issues: [iss-2610302057380175, iss-2610262034286775]
related_adrs: [adr-2610150603484802]
origin: researcher-authored
production_mode: hand-written
---

# The status line always names the facilitator's mode

## Press Release

> The status line and the bare board always name the technical facilitator's mode, human and auto alike, with whose setting decides it: the repository's, this computer's, or none set. Nia, a facilitator, works in two repositories. In the first the main branch says `auto`, and every row there says so and says it is that repository's. In the second nothing is set, and every row says human, not set; until now that row said nothing at all. When abcd has not read the mode, the row says it has not, and names none. Nothing about who merges changes: the row says what the settings already are.

## Why This Matters

The product thinker ruled on 2026-11-25 that the technical facilitator's mode becomes a personal setting and that the status line must say which it is; they called it critical and to be built without delay (dcn-2611252040395990). This is the second half of that ruling, and it needs no answer to the first. The row names the mode today only while it is `auto`, so a row that says nothing can mean human, not yet read, or a read that ran out of its share of the row's time, which happens on a busy computer (iss-2610302057380175). A person should not have to know which. The board does name it, under a label of its own (`merges:`), so the same fact has two names on two surfaces (iss-2610262034286775). It is built first so that the personal setting (itd-2611252049031692) is visible to everyone the day it lands.

## Mechanism

We expect a person to know from the row alone which mode the repository they stand in is on and whose setting makes it so, because the element is there in both modes, the part that costs git time is read ahead of the row and kept, and what was not read is called that. Shown wrong if abcd's row renders with no element; if it names a mode that the default branch as kept and the computer's file do not give; or if a person still opens the board to learn whether a silent row meant human.

## Scope Conditions

- Holds where abcd's row renders: a checkout abcd manages, on a host whose status line the person consented to, with the row not switched off.
- Holds for the settings there are: the default branch's committed `facilitator.mode` and the computer's file, resolved as the merge gate resolves them.
- Holds as of the last reading. The row reads the default branch as it was last read on this computer and asks no network; the merge gate fetches and reads for itself.
- Holds where the host runs abcd's prompt hook, which takes the reading. Where it does not, the row says the mode is not read.

## Acceptance Criteria

- **Given** a managed checkout and each of the nine pairings of the default branch (declares `human`, declares `auto`, declares nothing) with the computer's file (the same three), **when** the row renders, **then** directly after the branch it carries `facilitator:`, the mode the merge gate's resolver gives for that pairing, and whose setting decides it (the repository's, this computer's, or not set), and the words `autonomous merging` are gone. Table-tested over the nine, each naming both.
- **Given** a home that abcd refuses, so that the computer's file cannot be read, **when** the row renders, **then** it names `human`, as the resolver gives it, and says that the computer's file could not be read.
- **Given** a repository with no commit, **when** the row renders, **then** it reads `facilitator: human (not set)`. Every other case in which no mode can be read (a repository with commits whose default branch cannot be told, a setting that is refused, git not answering, no reading kept yet) reads `facilitator: not read`, and no mode beside it.
- **Given** a kept reading taken at the default branch's present tip, and a git that never answers a question about the mode, **when** the row renders, **then** the element is there with the kept mode, a test that counts the gits a refresh starts finds as many with the element as without, the row's 300 ms git share and 500 ms ceiling are unchanged, and stderr carries nothing about it. The kept reading is the one iss-2610302057380175 adds; whichever change lands first carries it.
- **Given** a kept reading, **when** the computer's file is written, changed, or removed between two refreshes, **then** the second refresh names the mode and the setting the file now gives: the computer's file is read at every refresh, and only the default branch's half is kept.
- **Given** a default branch that has moved since the reading was kept, **when** the next prompt reaches abcd's prompt hook and its read ends inside the one second the hook has for everything, **then** the kept reading is replaced and the next refresh names the new mode; **when** the read is late, or no prompt has come yet, **then** the row says `facilitator: not read` with the time of the reading it holds, and not that reading's mode, until a read ends in time.
- **Given** a setting that is refused, **when** the row renders, **then** the refusal is named on stderr once in that render, masked; a read that ran out of time is named nowhere but in the row.
- **Given** the kept reading, **when** it is written or read, **then** it is under abcd's home and never in a checkout, is the one checkout's and is never shown for another repository, and counts as no reading where it is a link, is owned by or writable by another account, is over its size, or is not valid; the status verb still writes its one fill file and nothing else, and the bare board still writes nothing (`commands/abcd.md`): neither of them keeps a reading. A file in the checkout, tracked or not, changes nothing the row says, and a test shows that no gate, guard, prompt line, or landing reads the kept reading.
- **Given** either mode, or one that cannot be read, **when** the bare `abcd` board renders in either view, as text or markdown, **then** its line is labelled `facilitator:` and not `merges:`, and names the mode, whose setting decides it, and what becomes of a ready pull request in the words the board has today; its presence line carries the same element, as the board itself reads it and not from the kept reading; and no line on the board calls this `mode` alone. Under `--json` the reading stays the `facilitator` object it is (`key`, `mode`, `layer`, `origin`, and `home_refused`, or `key` and `error`), and the row's element keeps its key, `merging`. Closes iss-2610262034286775.
- **Given** a `statusline.json` that switches `merging` off, **when** the row renders, **then** the file is accepted, the element shows all the same, and the switch that no longer does anything is named on stderr once in that render, as a switch for the badge is. The whole row's own off switch is unchanged.
- **Given** the change that ships this, **when** it lands, **then** the status verb's help, `commands/abcd.md`, `commands/mode.md`, and the board's and the mode's chapters of the brief say that the row names the mode as last read, that the gate does not rely on it, and that `facilitator.mode` is not the state `abcd mode` keeps.

## SOTA

Not searched for this draft: no source was opened, and the independent fit-challenge is owed before the build. Prompt and editor segments that show a mode at all times are the nearest field. Path 2: a native element of a row abcd already draws, with no dependency; the seam is the host's own status line, should a host come to show who decides.

## Open Questions

_None._

## Decisions

1. The product thinker's, on 2026-11-25 (dcn-2611252040395990): the status line says which mode it is; critical, and built without delay. They gave no wording for a criterion. The criteria, the Mechanism, and the scope conditions are the writing session's, for their review. To reverse: name the mode only while it is `auto`.
2. The session's, while the product thinker is away (dcn-2611252114265462): this is an intent of its own, split out of itd-2611252049031692, and is built first. To reverse: land the two together.
3. The session's, for review: the element loses its off switch, since the ruling says that the status line must say which it is. It puts away the decision of 2026-10-29 in itd-2610150602547271, made on the product thinker's delegation, that the row shows the mode only while it is `auto`. To reverse: keep the switch, on by default.
4. The session's, for review: the label on the board is `facilitator:`. It puts away `merges:`, the wording of dcn-2610230004263711. Two accepted answers of 2026-10-27 name the line differently (dcn-2610271112330536): iss-2610262034286775 says `facilitator:`, and iss-2610262036211949 gives `merges:` as its example. What the second asks for, that `mode` is left to `abcd mode`, is kept. To reverse: keep `merges:` on the board and use it on the row too.
5. The session's, no question put: the board keeps no reading, so that "Writes nothing" stays true of it without an exception. To reverse: name an exception on its page.
6. The session's, no question put: the row says `not read` and not what an unread mode counts as. What a gate does with a mode it cannot read is the gate's to say, where it says it. To reverse: add the words.
7. The session's, no question put: the element's key stays `merging`, since a file and a program may name it. The short words for whose setting it is are the builder's, within the criteria.

## Decomposition

Hand-run four-piece table, 2026-11-25, as the first part of a split made by the session (dcn-2611252114265462); a person confirms the routing. One capability, this intent: a person always sees the facilitator's mode and whose setting makes it so. No trust rule: nothing that decides a merge reads what the row reads. No new stance. The element, the kept reading, and the board's label are plumbing, for the spec and the brief's board and mode chapters (`04-surfaces/08-abcd.md`, `04-surfaces/26-mode.md`). Typed links: `builds_on` itd-2610150602547271, whose resolver and element this uses; `refines` iss-2610262036211949, the match made at filing, confirmed, since this names the line that issue is about and keeps its answer. Two reversals are flagged for a person to confirm, never auto-classified: in part itd-2610150602547271 (decision 3), and in part itd-2611031342000215, a draft whose rulings of 2026-11-03 show an `auto-merge` badge in `auto` alone; whichever is built second takes the other's reading. Verdict: FILE-AS-IS, as one part of the split.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
