---
id: spc-32
slug: abcd-update-completes-a-chosen-update-in-one-verb-it-fetches
intent: itd-130
---
# `abcd update` — fetch, verify, swap, in one verb

## Summary

spc-32 delivers itd-130: a user-invoked `abcd update [<tag>]` verb that
completes a chosen update for the PATH-installed binary — resolve (or accept)
the tag, fetch the platform asset over a pinned transport, verify it against
the same release's `checksums.txt`, swap atomically via `minio/selfupdate`,
and print a receipt. All logic lives in a new `internal/core/update` package
returning a structured report; `internal/surface/cli` renders it (adr-23).
Every design decision below is a grilled commitment carried from the intent's
Decisions section (2026-08-20 interview).

## Scope

- New package `internal/core/update`:
  - `Resolve(tag string)` — empty tag resolves `releases/latest` through the
    existing `internal/core/vintage` fetcher (one origin for checker and
    provisioner, per the release.go comment contract); a given tag is used
    verbatim after shape validation and termsafe sanitising for render.
  - `Plan(cwd)` — install-shape dispatch producing a typed verdict before
    any network fetch beyond resolution (see Dispatch below).
  - `Apply(plan, tag)` — download asset + `checksums.txt`, verify SHA-256,
    swap via `minio/selfupdate` (new dependency; sign-off recorded in
    itd-130's Decisions), emit `Report{Origin, Tag, Digest, OldVersion,
    NewVersion, Action, Refusal}`.
- CLI verb `abcd update` in `internal/surface/cli`: TTY detection
  (progress bar + resolved-tag confirmation on TTY; silent otherwise),
  receipt rendering in both modes, exit codes (0 applied or already
  current; non-zero refusal/failure, each named).
- Transport: a dedicated client in `internal/core/update` that (a) ignores
  `HTTPS_PROXY`/`HTTP_PROXY`/`ALL_PROXY` and `SSL_CERT_FILE`/`SSL_CERT_DIR`
  (explicit `Transport{Proxy: nil}` and a root pool built without env
  overrides), (b) follows redirects only to the release origin's own asset
  hosts, re-checked per hop under `internal/urlguard` (DialControl
  re-verification, as `vintage/release.go` does), (c) streams to a temp file
  in the destination directory under a size bound, (d) fsyncs before the
  swap. No body is ever read from an unverified redirect target.

## Dispatch (Plan verdicts)

Keyed on what actually runs: the first `abcd` PATH occupant (reusing
`internal/core/ahoy`'s `scanPathEntries`/`classifyBinTarget` seam — one
canonical ownership predicate; the ahoy package exports what update needs
rather than update re-implementing it).

| shape | verdict |
| --- | --- |
| binary resolves into a plugin root | refuse; name the host's plugin-update path (itd-108 one-cut coherence) |
| dev shim at the entry | refuse; name the shim's track-latest contract and `ahoy install` mode switch |
| abcd-owned dangling symlink (iss-345 shape) | refuse; name `abcd ahoy install` as the heal |
| owned symlink, healthy | refuse; the plugin update owns that binary — name the plugin-update path |
| regular file whose SHA-256 is proven against a published release (the install tag's manifest first, then the release list walked newest-first from the origin's atom feed, bounded) | OWNED BY PROVENANCE → swap in place; the old version is the release the on-disk bytes belong to, derived here, never read from the running binary |
| regular file matching no release | refuse; describe the occupant, suggest `--help` install docs |
| resolved path under a brew prefix (`/opt/homebrew`, `/usr/local`, `/home/linuxbrew/.linuxbrew` — prefix test on the RESOLVED path) | refuse; print `brew upgrade abcd` |
| entry shadowed by an earlier PATH occupant | proceed on the owned entry but the receipt reports the shadowing occupant; never claim the machine runs the new version |
| nothing on PATH | refuse; name the install one-liner and `ahoy install` |

Every refusal is a structured `Refusal{Shape, Detail, Remedy}` — loud, named
remedy, non-zero exit (per the intent's refusal criteria).

## How each acceptance criterion is satisfied

1. **Verified swap + receipt, exit 0** — `Apply` verifies the asset digest
   against same-release `checksums.txt` before `selfupdate.Apply`; `Report`
   carries origin/tag/digest/old→new; CLI exits 0.
2. **Checksum mismatch** — digest compare failure aborts before any write to
   the target path; both digests in the error; non-zero exit. Test: serve a
   mismatched pair through the test transport seam.
3. **Hostile proxy/CA env** — the transport is constructed with `Proxy: nil`
   and a system root pool loaded without `SSL_CERT_FILE`/`SSL_CERT_DIR`
   (t.Setenv tests assert both are ignored: the request goes to the test
   server directly and env-planted CA files are never read — asserted via a
   canary file whose read would be observable).
4. **Plugin-root refusal** — `Plan` short-circuits before any fetch; test
   with `ABCD_PLUGIN_ROOT` pointing at a hermetic root.
5. **Dev-shim refusal** — `classifyBinTarget` → `binTargetDevShim` verdict;
   test reuses ahoy's shim fixtures.
6. **Shadowing report** — `Plan` records `shadowingEntry` output into the
   report; test mirrors ahoy's shadowed fixtures.
7. **No network → loud fail, no partial file** — resolution failure surfaces
   the vintage fetcher's bounded error; download failure unlinks the temp
   file (defer-cleanup asserted by test walking the target dir).
8. **Zero-network seam** — the release-origin client is constructible only
   inside `internal/core/update` and `internal/core/vintage` (package-var
   seam like `version.go`'s `newReleaseFetcher`, so the existing
   zero-network harness proves no other verb reaches it).
9. **Non-TTY silence** — progress writer selected on `term.IsTerminal`;
   non-TTY test asserts stdout carries only the receipt.

## Deliberately out (carried from the intent)

Ambient checks (adr-38), the Homebrew tap, plugin-surface delivery
(itd-108), the bootstrap trampoline demotion (follow-on rung; delegation
seam pre-specified in the intent's Decisions), in-process attestation
(deferred; checksums bar per itd-105), Windows cold-start shim. Sequencing:
iss-232's `vcs.revision` guard lands before the first release this verb can
fetch.

## Test-first order

Each dispatch row and criterion lands as a failing test before its code;
the transport policy tests (proxy/CA/redirect) precede the transport; the
`minio/selfupdate` dependency enters with the first swap test
(`Assisted-by` and dependency sign-off already recorded).
