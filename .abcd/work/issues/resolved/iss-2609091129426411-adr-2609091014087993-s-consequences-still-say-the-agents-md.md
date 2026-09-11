---
schema_version: 1
id: "iss-2609091129426411"
slug: "adr-2609091014087993-s-consequences-still-say-the-agents-md"
severity: "minor"
category: "documentation"
source: "impl-review"
found_during: "recfix-adversarial-review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/decisions/adrs/2609091014087993-a-tool-never-creates-directories-in-user-owned-project-space.md"
resolution: "Superseded by adr-2609091248200336, which states the split: the location half of the worktree rule binds now (AGENTS.md § Concurrent sessions names the store, 805bb023) and the verb half binds when itd-2609091014076309 ships. The original's decision text is untouched; its status and forward link mark it superseded."
impact: internal
---

adr-2609091014087993 says two things about `AGENTS.md` that cannot both be read at face value. Among its normative clauses, in the present indicative: "`AGENTS.md`'s concurrent-sessions convention names the store as the place a session's worktree goes, and names no sibling-directory form." Among its Consequences: "The concurrent-sessions convention gains the store as the place a worktree goes, and loses any wording that reads as a sibling directory. That edit lands with the intent, because a convention naming a verb that does not exist is a phantom gate." The first states as fact what the second schedules, and when the ADR was accepted neither was true of the file: § Concurrent sessions was byte-unchanged, named no store, and treated creating a worktree inside the checkout as ordinary — the shape the ADR rejects as its alternative 2.

An ADR is never amended, always superseded, so the contradiction was closed from the other end on 2026-09-09: § Concurrent sessions now names `~/.abcd/worktrees/<root-sha>/<name>/` as where a session's worktree goes, gives the ADR and the principle as its grounds, refuses the sibling and in-checkout forms, and says in as many words that the store has no verbs — a plain `git worktree add` aimed at the path, with nothing to enumerate or prune the lane until itd-2609091014076309 ships. That makes the normative clause TRUE while honouring the reason the Consequence gave for deferring, which is about naming a VERB that does not exist and not about naming a location that plain git already reaches. The principle `the-users-directory-is-theirs` already stated the same hand-followed rule, so the convention is now consistent with it rather than silent.

What is owed is the residue: the ADR's Consequence bullet reads as though the whole edit is still ahead, when its store half is behind and only its verb half remains — the intent's acceptance criterion that § Concurrent sessions "names the verb as the way a session gets its own checkout". Nothing in the record says the edit lands in two steps, and a reader of the ADR alone would look for an unmade change.

DECISION NEEDED, and it is the maintainer's because it is an ADR. Options: (a) accept the bullet as a prediction fulfilled in two steps and let the intent's shipping close it, recording that reading in DECISIONS.md; (b) supersede adr-2609091014087993 with a record that states the split explicitly — the location half binds now, the verb half binds when the store ships; (c) revert the AGENTS.md edit and hold both halves for the intent, which restores the contradiction the ADR carried when it was accepted and is only defensible if naming the store without a verb is judged a phantom gate after all.

Detector: none is possible in code. The enforcement-claims-are-facts principle names the promotion that would catch this class — a record-lint rule cross-checking a named gate, convention or file against what that file actually says — and none exists.

## Grounds

- pursued: a reader of the record alone now finds one statement of what AGENTS.md says and what is still owed, so nobody goes looking for an unmade change; shown wrong if the next reviewer of adr-2609091248200336 or of itd-2609091014076309's last criterion still reads the AGENTS.md edit as wholly ahead, or if the intent ships without the verb clause landing in the convention
