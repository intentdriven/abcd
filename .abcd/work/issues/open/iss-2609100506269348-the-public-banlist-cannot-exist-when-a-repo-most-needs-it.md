---
schema_version: 1
id: "iss-2609100506269348"
slug: "the-public-banlist-cannot-exist-when-a-repo-most-needs-it"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-07/08; re-filed into abcd 2026-09-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal (ahoy gitignore policy, banlist public layer)"
related_intents: [itd-2609151516525843]
deferred_after: "v0.9.0"
deferral_reason: "Ruled by the product thinker at the 2026-09-23 run A interview (M27: fold itd-2609151516525843 into the older draft itd-159 and plan them as one intent carrying both halves ruled 2026-09-15; the switch-value-or-exception ruling is taken in that planning interview). Earlier deferral: The public banned-names list cannot be created on a fresh public repository, and the cause is a bootstrap paradox rather than a bug: the visibility fence is narrowed only on positive evidence that the record directory is committed, and the fence prevents that evidence from ever existing. Every route runs through what public visibility is declared to mean, which is a documented contract pinned as a literal. An earlier record already ends with three candidate reconciliations for a maintainer to pick between, and the intent it was promoted into is still an unfilled draft. Picking one is the product thinker's call."
---

On a fresh PUBLIC repo the committed banned-names layer cannot be created, and the window in which it cannot is exactly the window in which a repo is being set up to ban a name.

The mechanism, which abcd states clearly once you are in it. For visibility `public`, `ahoy install` writes a `.gitignore` fence of `/.abcd/` — the whole namespace. `banlist add --public` then refuses, because `.abcd/docs-lint.json` would be ignored and "a config written there would reach no CI run". The fence narrows to `.abcd/.work.local/` only once `.abcd/` already holds TRACKED files, because narrowing "needs positive evidence". A brand-new repo has none, so the fence stays wide, so the public layer cannot be written.

The escape exists and is not discoverable: commit the record tiers first with `git add -f` against the tool's own fence, then re-run `ahoy install`, which narrows the fence and only then writes the store. Three steps, one of them forcing past a gitignore the tool just wrote, none of them named by the refusal. The fix hint points at iss-176 and offers "commit the path explicitly, or ban the name on the private layer instead"; the second silently drops CI enforcement, which is the whole point of the public layer.

Why this is worse than an ordering wrinkle: the reason a repo reaches for the public banlist on day one is that it has a name it must never publish. That is precisely a rename or an extraction, which is precisely a new repo with nothing tracked under `.abcd/` yet. The guarantee is unavailable at its own moment of need, and a maintainer who takes the offered second option gets a ratchet that CI never enforces without being told that is what changed.

Needed: let `ahoy install` write the public store and narrow the fence in one pass on a repo it is adopting (it is writing both files anyway), or refuse the `public` fence entirely for the record tiers, which abcd's own repository already does by committing `.abcd/` and fencing only the local tier.

Adjacent to iss-223, which reports the same fence hiding already-committed records on a public repo; this is the other end of it, the fence preventing a record tier from ever becoming committed.

## Grounds

- pursued: the bootstrap paradox closes on a committed DECLARATION rather than on detected evidence — narrowing the fence waits on tracked files under the record namespace and the fence is what stops them existing, while a declaration is evidence a repository can give on its first commit — and the second half closes on scope: the names a person must never publish belong to that person and their machine rather than to any one repository, so a machine-global private list in the user-level home bans them everywhere at once. What would show it wrong: a fresh public repository that still cannot create its committed list with the declaration present; a machine-global entry that fails to ban a name in a second repository on the same machine; or either layer crossing the other's boundary — the home list read by CI, or any of its patterns reaching a committed file.
