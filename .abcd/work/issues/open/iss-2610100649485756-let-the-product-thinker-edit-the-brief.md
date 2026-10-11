---
schema_version: 1
id: "iss-2610100649485756"
slug: "let-the-product-thinker-edit-the-brief"
severity: "minor"
category: "future-work-seed"
source: "managed-repo"
found_during: "abcd inbox report rpt-2610071632391737 from a managed repository (root commit c372ff6b8fc387ffdf1acf7c51bffe469b70b2e0)"
origin: researcher-authored
production_mode: hand-written
found_at: "brief, intent records, a new edit verb, the review verb proposed in rpt-2610071611295053"
remedy: "none (filed automatically)"
related_intents: [itd-2610040740122709]
---

Let the product thinker edit the brief and their parts of intents in a local web editor before anything is committed

## The need

A product thinker owns the brief and the user-facing parts of intents, but today editing them means a terminal editor or the agent writing for them. In an agent session neither works well: the session's shell has no terminal, so a VISUAL or EDITOR set to a terminal editor (here: emacs -nw) cannot open, and no editor server was running. The owner asked for an option to edit the brief, and their parts of intents, before anything is committed to GitHub.

## Design (discussed with the owner)

1. **Editor resolution:** an abcd setting in user-level config, then VISUAL, then EDITOR, then the platform default. At a terminal, block like git commit; in an agent session, use a non-blocking form the person configured, or print the exact command for their own terminal; never fail silently.
2. **A self-served local web editor:** abcd starts a loopback-only server (random port, single-use token carried in the URL fragment, Host header check against DNS rebinding, lives for one round) and serves a verified copy of the editor release. The owner's domain (static hosting, no accounts, no data) only distributes releases. Same origin avoids CORS, the local-network-access prompt and mixed-content limits, and the domain cannot act on private files at run time.
3. **Record-aware editing:**
   - **Brief:** fully editable.
   - **Intents:** press release, why it matters, acceptance criteria, mechanism, scope condition text (identity markers hidden and preserved) and open questions/decisions editable.
   - **Through verbs only:** grounds through a form calling the verb; plan, hold and target as confirmed buttons (plan is the sign-off act).
   - **Read-only:** ids, links, status fields and audit notes.
4. **Checks:** deterministic checks on every save (name guard, record shape, links), judgement checks on done (conflicts with the vision page's tenets; the brief-to-intent consistency pass, flagging an edit that contradicts a filed, planned or shipped intent as a design decision owed).
5. **Before commit:** the session's changes per file, each kept or discarded; discard restores the session-start text, never HEAD, so earlier uncommitted work survives. Commit is a separate act, on a branch, under the person's own identity; push only when asked.

## Evidence coming

The lab is building a prototype of 2 to 5 (loopback, self-served, token, record-aware intent form, lint on save, per-file keep/discard, a timed event log) to test with the product thinker in this repository. Its session logs will be attached to a follow-up report.

Remedy the reporter proposes: Add an edit verb that serves a record-aware editor from a local loopback server (release code distributed from the owner's domain and checksum-verified, like the binary), runs abcd's checks on every save and the judgement checks on done, shows the session's changes per file for keep or discard, and leaves the commit as a separate act under the person's own identity.

Reported by a managed repository (root commit c372ff6b8fc387ffdf1acf7c51bffe469b70b2e0) through the abcd inbox as rpt-2610071632391737, a enhancement against abcd v0.13.1, surface brief, intent records, a new edit verb, the review verb proposed in rpt-2610071611295053.

Evidence:

- rpt-2610071632391737 (the report, kept in the inbox)
- lab-261002171217-c372ff6
- rpt-2610071611295053
- rpt-2610071608595198
