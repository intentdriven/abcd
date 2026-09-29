---
schema_version: 1
id: "iss-2609290703091174"
slug: "open-question-for-the-product-thinker-should-abcd-refuse-to"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil/home.go"
remedy: "Waits on ruling BO (CA7); in internal/fsutil/home.go: if refused, reject a ~/.abcd whose mode has 0o020 and change the documented setup to mkdir -m 700 ~/.abcd; if accepted only for a private group, admit 0o020 only when the directory's group has its owner as sole member, as Debian's OpenSSH does; if kept, record why beside the stricter readings of the declaration file and ensureStore so the three agree by statement. Prove the answer with a mode-table test over 0755, 0775 and 0777."
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (lane drainFresh of autonomous run A, 2026-09-29): Should abcd refuse to read its settings from a ~/.abcd folder its owner's group can write (0775)? Refusing protects a machine whose group is shared, and newly refuses a hand-made folder on Debian and Ubuntu, where the default umask makes one; today the declaration file and the worktree store already refuse it while the settings read admits it."
---

Open question for the product thinker: should abcd refuse to read its own settings from a ~/.abcd folder that the owner's group can write to? Today it refuses a folder every account can write, and one another account owns, but reads from a group-writable one (0775). Refusing would protect a machine where the group really is shared with other people. It would also newly refuse ordinary installs on Debian and Ubuntu, where the login umask for a user whose group is private to them is 002, so the documented 'mkdir -p ~/.abcd' makes a 0775 folder whose group is only its owner. abcd's own writers create ~/.abcd at 0755 or 0700, so only a folder made by hand is affected. Two readings already disagree: the declaration file itself is refused when group-writable, and the worktree store (implement/loop ensureStore) refuses a group-writable ~/.abcd level. Deferred out of iss-2609290656480443's fix until ruled.

## Remedy grounds (2026-09-29)

- Why: the three answers CA7 lists, each with its change; the ruling is unanswered and none is picked.
- Sources (consulted 2026-09-29): sshd_config(5) StrictModes checks the modes of the user's files and home directory (https://man.openbsd.org/sshd_config), and upstream OpenSSH refuses any mode with 022 set; Debian's user-group-modes patch admits group-writability 'provided that the group in question contains only the file's owner' (https://sources.debian.org/patches/openssh/1:9.2p1-2+deb12u7/user-group-modes.patch/). The private-group answer is the one the field has shipped for exactly the Debian and Ubuntu umask case this record names.
- Rejected: reading the umask at run time, which says how the folder would be made, not who can write it now.
