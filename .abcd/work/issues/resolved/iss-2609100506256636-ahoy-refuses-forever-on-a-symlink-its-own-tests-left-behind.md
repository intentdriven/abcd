---
schema_version: 1
id: "iss-2609100506256636"
slug: "ahoy-refuses-forever-on-a-symlink-its-own-tests-left-behind"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-07/08; re-filed into abcd 2026-09-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal (ahoy install, symlink.foreign)"
resolution: "A link at the PATH target that resolves to nothing is now a repairable gap rather than a permanent refusal. The discriminator is danglingness, not provenance: a link that resolves to a file is somebody's working install and is still refused untouched, while a link that resolves to nothing runs nothing, cannot be anyone's install, and is cleared. abcd does not try to prove it wrote the link, because it cannot, which is why the state was unreachable. The write-through hazard is closed structurally, since the repair removes the link rather than writing through it. The wording for both dangling shapes now comes from one builder so the two cannot drift, and the non-owned case asserts no provenance. A live foreign link keeps its refusal and has a negative control asserting the link is left byte for byte."
impact: fix
---

abcd's own test suite leaves a dangling symlink on PATH, and `ahoy install` then refuses to complete for ever, with no supported way to clear it.

Found at the very start of adopting abcd in a managed repository. `abcd` was not on PATH at all, and the user-level bin entry was a symlink pointing into a deleted temp directory whose name is a Go test name from abcd's own ahoy install suite (a `TestAhoyInstallAccepts…` directory, with the numbered subdirectory a t.TempDir tree leaves behind). So a test run wrote a real symlink into the user's real bin directory and did not clean it up.

The consequence is worse than the litter. That directory is early on PATH, so every bare `abcd …` resolved to a broken link, including the command abcd's own SessionStart hook tells the operator to run. Detection reported it correctly as `symlink.foreign` and refused: "Resolve manually; ahoy refuses to clobber." That refusal is right in general — abcd must not delete a binary someone else installed — but the state is unreachable from inside the tool: there is no `--force`, no `ahoy uninstall` path that removes a foreign entry, and the gap is `resolvable: false`, so `ahoy install` can never finish. The only way out is a manual removal the operator has to be told to run.

Two things worth separating. Refusing to clobber a foreign symlink is correct. Refusing to clobber a symlink that is DANGLING is not obviously correct: a link whose target does not exist runs nothing, shadows the real binary, and cannot be anyone's working install. abcd already reads the target to report it, so it already knows the target is missing.

Needed: (1) the test suite must not write into the user's real bin directory — point it at a temp bin dir; (2) treat a dangling symlink as replaceable, or offer an explicit override that names what it is replacing, so a repo can be adopted without shell surgery. Both are small; the first is the one that created the problem.

## Grounds

- pursued: we expect danglingness to be the right discriminator because a link resolving to nothing cannot be in use by anyone, so clearing it takes nothing from a user while ending a permanent wall; it is shown wrong if a dangling link is ever a deliberate placeholder someone intends to fill
