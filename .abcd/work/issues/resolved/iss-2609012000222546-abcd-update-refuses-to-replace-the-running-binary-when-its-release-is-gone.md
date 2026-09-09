---
schema_version: 1
id: "iss-2609012000222546"
slug: "abcd-update-refuses-to-replace-the-running-binary-when-its-release-is-gone"
severity: "major"
category: "process"
source: "user-observation"
found_during: "abcd-update-invocation-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/update/update.go"
resolution: "abcd update proves ownership without the forge — the running executable's own identity, or the ~/.abcd/path-entry install record — and the refusal that remains reinstalls over the file instead of sending the user to delete it"
impact: fix
---

abcd update refuses to replace the very binary that is running it when that binary's release no longer exists on the forge, and its remedy sends the user to remove the file, which removes the only tool that could reinstall. Observed 2026-09-01: ~/.local/bin/abcd was v0.6.6 (byte-identical to the plugin-cache binary the bootstrap provisioned); abcd update resolved v0.7.0, then refused with shape unprovenanced-file because the file's digest appears in no published checksums.txt. The 2026-08-30 decision deleted every release object older than v0.6.9 and notes that v0.6.0-v0.6.6 never had one, and it claims removing old assets affects only a deliberate pin; that is wrong for abcd update, whose ownership proof (spc-32, the OWNED BY PROVENANCE row) walks exactly those deleted manifests, so every v0.6.7, v0.6.8 and plugin-provisioned install now refuses the same way. The remedy text ('remove it and reinstall') names no reinstall command, and once the file is removed the update verb cannot run at all; the user was left with no abcd and recovered only by the README one-liner. Two directions, neither adopted: (1) when the target resolves to the running executable itself (os.Executable after symlink resolution, os.SameFile), the file is abcd by construction and cannot be a foreign binary, so provenance should only derive the old version, reporting it as an unpublished build with its digest and proceeding after the existing TTY confirmation; (2) at minimum the refusal remedy carries the reinstall one-liner or the docs path and states that update cannot run once the file is gone. Part of the ease-of-update design work: users must be able to move the CLI and the plugin route forward without a dead end.

Amended 2026-09-09 on the maintainer's ruling: pre-1.0.0 binaries are out of
scope, and what matters is that updates work going forward. No rescue is built
for the historical population. What is fixed is the MECHANISM the dead end came
out of — an ownership proof resting on release objects that can be deleted.
Ownership is now proven first by the running executable's own identity
(os.Executable, symlinks resolved, os.SameFile: the bytes that loaded the code
asking the question cannot be a foreign binary) and then by the
~/.abcd/path-entry install record, neither of which any release deletion can
revoke; the manifest walk is demoted from proving the file to dating it, and a
file it cannot date is reported as an unpublished build with its digest.
Direction (2) is taken as well: the refusal that remains reinstalls OVER the
file, names the one-liner and `ahoy install`, and states that the verb cannot
run once the file is gone. What this does NOT fix, deliberately: an abcd already
installed at v0.6.x carries the old code and still refuses, because the fix
lives in the binary being replaced. Its route back is the README one-liner —
which is how the 2026-09-01 reporter recovered.

## Grounds

- pursued: an ownership proof rooted in this machine rather than in deletable release objects keeps abcd update working across any future asset cleanup; a refusal on a binary abcd itself installed, or a user again told to remove their only copy, would show it wrong
