---
schema_version: 1
id: "iss-95"
slug: "wiring-the-stop-hook-does-not-by-itself-start-the-transcript"
severity: "major"
category: "architectural-insight"
source: "manual-test"
found_during: "itd-89-m1"
found_at: "internal/surface/cli/cli.go"
resolution: "The transcript store is user-level, self-creating and keyed on the root-commit SHA: ~/.abcd/transcripts/<root-sha>/{records,staging}/. Capture no longer has an install precondition, so the SessionEnd hook on a machine where ahoy install never ran stores a transcript instead of logging a line and exiting 0. The per-repo location is an opt-in pull declared in ~/.abcd/local-transcript-roots (home-scoped, uid-owned, not other-writable), landing in the gitignored .abcd/.work.local/transcripts/. A corpus at ~/.abcd/history/<root-sha>/transcripts/ is moved into the store on first resolve, reported on stderr, and tombstoned at the old path."
impact: fix
---

Wiring the Stop hook does NOT by itself start the transcript clock: history.Capture requires ~/.abcd/history/<root-sha>/transcripts/ to already exist and deliberately never creates it (ownedDirsReal validates the store's dirs are real, not symlinks). That dir is bootstrapped by 'abcd ahoy install'. On a machine where install has not run — including this one, where ~/.abcd/ does not exist at all — 'hook session-end' fails closed, logs to stderr, exits 0, and captures NOTHING. Silently. That is precisely the failure mode itd-89 exists to prevent: a hook that appears wired while the corpus never accrues. Decide: (a) the hook bootstraps the store itself (changes Capture's stated precondition, and a hook creating dirs is a trust-boundary act the ownedDirsReal discipline deliberately avoids), or (b) 'ahoy install' stays the sanctioned bootstrap and the not-installed case is made LOUD rather than a stderr line nobody reads (e.g. ahoy doctor already flags history.bootstrap_missing as a required gap). Until this is settled, itd-89's acceptance is met in code but not on any machine that has not installed.

## Grounds

- pursued: relocating the store out of the install-owned registry namespace and making it create itself is expected to dissolve the silent no-capture rather than report it, because nothing is left between a wired hook and a stored transcript; a session that ends with nothing staged or stored on a machine that never installed, or a corpus stranded at the old path, would show it wrong.
