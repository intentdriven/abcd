# Durable state lives where the platform says it survives

**The rule.** State that must outlive an event the platform owns — an update,
a re-clone, a garbage-collection sweep — is stored only in a location the
platform *documents* as surviving that event. Never fight the host lifecycle
inside a directory the host replaces or deletes on its own schedule, and never
guess at a survivable location the platform did not hand over: a wrong guess
plants trusted state in an untracked place, which is worse than losing it.

**Why.** The 2026-08-21 plugin-update post-mortem (iss-2608210934566221)
priced the violation precisely: the downloaded hook binary lived in the
harness's commit-stamped plugin directory — re-cloned on every update,
documented as not preserving extra files, garbage-collected roughly fourteen
days after being orphaned. One routine update cost a full re-download, a
first-hook window with no binary, one binary copy per plugin version left for
the sweeper, and a PATH symlink two weeks from dangling
(iss-2608210934566222). None of that was a bug in abcd or in the harness: each
side behaved as documented, and the loss was the storage decision itself. The
same harness documents a persistent per-plugin data directory that survives
updates and dies only on full uninstall — the platform-sanctioned home the
binary cache moved to (itd-132 / spc-35, seeded as iss-2608210934566227).

**Bounds.**

- The warrant is the platform's *documentation*, not observed behaviour: a
  directory that happens to survive today is not a home, and a documented home
  is taken on its documented terms (deleted at uninstall means gone at
  uninstall — plan for that, not against it).
- The rule cuts both ways: transient, per-version state (the executing binary
  paired with its plugin surface) rightly stays in the per-version directory;
  only what must *outlive* the lifecycle event moves out.
- Location changes trust obligations, never trust bars: state promoted into a
  longer-lived home keeps every verification it needed in the short-lived one,
  plus whatever the longer at-rest window now demands
  ([adr-46](../decisions/adrs/0046-persistence-never-weakens-the.md)).
- When the platform hands over no survivable location, degrade loudly to the
  transient behaviour — deriving the undocumented path is the guess this rule
  forbids.

**Promotion.** Unenforced. A lint that flags writes of long-lived artefacts
into known-transient roots (the plugin root, temp dirs) would promote this to
a discipline-kind intent; until then it is applied at review time.
