# The user's directory is theirs

**The rule.** A user is never surprised by a directory filling with things
they did not make. A tool creates files and directories only in space that was
handed to it — its own machine-scoped home under `~/.abcd/`, or the tiers of a
checkout the record declares — and never beside the user's projects, in a
parent directory, or anywhere the user did not point it. What it puts in its
own space, it can list and reclaim.

**Why.** A directory is the user's map of their own work: what is there is
what they made, and a folder that fills on its own stops being a map. The
2026-09-06 session that created twenty-two worktrees beside the maintainer's
other projects did nothing wrong by isolating — parallel sessions need
separate checkouts — and everything wrong by location; the objection was not
"why so many" but "I don't want a user to be surprised that a folder is all of
a sudden full of stuff". Twenty-one spent worktrees had already been cleared by
hand five days earlier (iss-2609020721142452). An agent works unattended on
trust, and a surprise in the user's own directory is what spends it.
[adr-2609091014087993](../decisions/adrs/2609091014087993-a-tool-never-creates-directories-in-user-owned-project-space.md)
records the ruling; this file carries the stance.

**Bounds.**

- Isolation is not the violation. The checkout is the unit of isolation
  (`AGENTS.md` § Concurrent sessions) and a parallel session needs its own;
  the rule governs where that checkout lands, not whether it exists.
- The checkout is handed over on the record's terms. `.abcd/.work.local/` is
  scratch by declaration and the repository root is not — a file guessed onto
  the root is a `stray_root_docs` finding — and beside the checkout there is no
  declared tier at all.
- A directory the user made by hand is theirs even inside the tool's own
  store: it is listed, never moved, never removed. The store deletes only what
  it can prove it created, the stance embark's destination gate already takes
  ([adr-35](../decisions/adrs/0035-lifeboat-as-coverage-experiment.md)).
- Adjacent, not the same:
  [`durable-state-lives-where-the-platform-says-it-survives`](durable-state-lives-where-the-platform-says-it-survives.md)
  chooses a location by what outlives the platform's lifecycle; this rule
  chooses by whose space it is. A sibling directory survives every platform
  event and still belongs to the user.

**Enforcement.** None on the write side. The read side exists: the loader
refuses a configuration root the caller does not own, and only
`~/.abcd/trusted-roots` re-admits it. The store this rule points at — a
root-SHA-keyed `~/.abcd/worktrees/` lane, a verb that lists it, a verb that
reclaims a merged worktree, a line on the status board — is
[itd-2609091014076309](../intents/drafts/itd-2609091014076309-session-and-agent-worktrees-live-in-a-machine-scoped-store-t.md),
in drafts. Until it ships the rule is applied by hand: a session that needs a
worktree puts it under `~/.abcd/worktrees/<root-sha>/<name>/`, a verifier's
copy goes to `.abcd/.work.local/scratch/`, and a reviewer who sees a directory
appear beside a checkout names it.

**Promotion.** The intent's verbs promote the store half; a check that counts
a repository's worktrees outside the store, and a hazard-registry entry that
makes `abcd guard` refuse a `git worktree add` aimed at a parent directory,
would make the rule checkable and promote it to a discipline.
