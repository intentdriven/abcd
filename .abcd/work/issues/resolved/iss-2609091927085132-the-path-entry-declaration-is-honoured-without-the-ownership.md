---
schema_version: 1
id: "iss-2609091927085132"
slug: "the-path-entry-declaration-is-honoured-without-the-ownership"
severity: "major"
category: "security"
source: "review-followup"
found_during: "v0.8.0 release gate, brief-surface crosscheck"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/owned_copy.go"
deferred_after: "v0.7.1"
deferral_reason: "The scope of what `~/.abcd/path-entry` is trusted to assert is a decision already recorded in DECISIONS.md and disclosed to users in the v0.8.0 notes: it is ownership against a hijacked PATH, not verification against a compromised account. Extending it to refuse a file another uid can write is a widening of that decision, not a defect against it, and widening it belongs to the maintainer rather than to the release that is mid-tag. Nothing user-facing claims the stronger property: the brief did, and that false claim is corrected in this same release. Deferred for one cycle so the decision is taken deliberately; the waiver lapses at v0.8.0 and the finding returns to the gate."
resolution: "A declaration naming which binary the hooks may execute is now honoured only when this session owns it and nobody else can write it. Four live read sites accepted a group-writable or foreign-owned record, and every one of the five hooks executed an attacker-named binary in the proving test; the presence test also followed a symlinked record. One canonical primitive beside the existing guarded read now checks link, regular file, mode and owner before reading, with a refusal enum so each caller keeps its own wording, and the two existing copies were migrated onto it rather than a third being added. The shell shims test the same properties with one file test rather than parsing a listing, which avoids field-position, locale and access-control-suffix assumptions and catches the symlink shape, and fails closed when its tools are missing. The accepted residual that a home-scoped record is writable by the same account stands untouched and is stated in the code, as does the parent directory's own mode, whose closure is a widening across all three declaration files rather than a defect in this one."
impact: fix
---

`~/.abcd/path-entry` decides which binary the hook shims execute. It is read
without the two checks that the other two home-scoped declaration files require,
and it is the one of the three where the consequence is code execution rather
than a location choice.

## What the siblings do

`~/.abcd/trusted-roots` and `~/.abcd/local-transcript-roots` are each read behind
the same three-part guard, on the stated ground that "a declaration this process
does not own, or one anyone can write, is not the caller's word and re-admits
nothing":

- a regular file (`Lstat`, not a symlink or device),
- `Perm()&0o022 == 0`, so not writable by group or other,
- `ownerUID(path) == os.Getuid()`.

See `internal/core/rules/root.go:300-311` and
`internal/core/history/location.go:222-231`.

## What path-entry does

`readPathEntry` (`internal/core/ahoy/owned_copy.go:99`) reads it through
`fsutil.ReadGuarded`, which enforces `O_NOFOLLOW`, regular-file and a size cap
and nothing else: no owner check, no permission check. The shell rung in
`hooks/hooks.json` is weaker still. It carefully establishes that the candidate
binary resolves absolutely, does not live inside the working tree, and sits in a
directory that is not world-writable, and then parses the file that names that
binary with a bare `while IFS= read -r ln` loop, with no check on the file at
all.

## Why it is not covered by the recorded residual

`iss-2609012039107700` records the accepted residual as "home-scoped and
same-UID writable", and the v0.8.0 changelog repeats it. That covers an attacker
who already has the user's uid. It does not cover a `path-entry` that is
group- or other-writable, which lets a *different* local uid name a binary of
their choosing in a directory they own at mode 755: every shell check passes,
and the hook executes it on every prompt, tool call and compaction. That is the
case `Perm()&0o022` exists to refuse, and the two siblings do refuse it.

## Shape of the fix

One canonical primitive rather than a fourth bespoke copy: the three-part guard
is currently written twice, in `rules` and in `history`. Lift it into `fsutil`
beside `ReadGuarded` as the shared declaration-file read, use it from
`readPathEntry`, and migrate the two existing callers onto it. The shell shims
need the same two facts checked before the file is read: `ls -ld` gives both the
owner name to compare against `id -un` and the mode string to test for group and
other write.

## Acceptance

- **Given** a `~/.abcd/path-entry` that is group- or world-writable, **when** any
  shim resolves an `abcd` on PATH, **then** the record is ignored and the shim
  takes its existing loud refusal path.
- **Given** a `~/.abcd/path-entry` owned by another uid, **when** the Go reader
  loads it, **then** it reports not-ok exactly as a truncated record does.
- **Given** a correctly owned, correctly permissioned record, **when** either
  reader loads it, **then** behaviour is unchanged from v0.8.0.

## Grounds

- pursued: we expect ownership and write-exclusivity on the declaration to be the property that matters, because the declaration's whole job is to vouch for a binary and a record anyone can rewrite vouches for nothing; it is shown wrong if the directory holding it is writable by another account, which this does not close
