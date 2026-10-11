---
id: spc-21
slug: a-fresh-plugin-install-just-works-the
intent: itd-105
---
# a-fresh-plugin-install-just-works-the-hook-binary-survives

## Summary

spc-21 delivers itd-105: a committed POSIX-sh bootstrap, wired as the first
`SessionStart` hook, that provisions `$CLAUDE_PLUGIN_ROOT/abcd` from the
latest GitHub release — checksum-verified, atomic, no-op when the binary is
already present — so a fresh marketplace install and every subsequent plugin
update yield a working hook surface with no manual step. The binary is shell
plus wiring plus reporting: no `internal/core` behaviour changes.

## Scope

- **`hooks/bootstrap.sh`** (new, committed): the self-provisioning script.
  Needs no abcd binary to run — the binary is exactly what is missing.
- **`hooks/hooks.json`**: the `SessionStart` event is ONE entry whose command
  runs the bootstrap and then the binary-backed `prompt-router-reset` /
  `session-start` calls in the same shell (see Hook wiring, and the correction
  recorded there).
- **`$CLAUDE_PLUGIN_ROOT/.binary-meta`** (new, written by the bootstrap): one
  small key=value file recording `release_tag`, `release_sha` (when
  resolvable), `fetched_at`, and `plugin_sha` — the input for skew reporting.
- **`internal/core/ahoy/guard_health.go`**: `BinaryReachable` already probes
  `pluginBinaryPath`; extend the rendered guard-health detail so a missing
  plugin-root binary names the bootstrap and its recovery step.
- **`internal/surface/*` session-start rendering**: the one-line version-skew
  notice (below).
- **`README.md`** plugin section: the contract — the plugin self-provisions
  its binary; the PATH install remains for terminal CLI use; build-from-source
  (`go build ./cmd/abcd`) is the documented full-trust path.
- **CHANGELOG entry** (user-facing behaviour change).

Out of scope per the intent: release pipeline, auto-update policy, signature
infrastructure, Windows, one-notice-per-session UX (iss-168 family), core
logic changes.

## Approach

### The bootstrap script

Behaviour, in order:

1. **Fast path.** `[ -x "$CLAUDE_PLUGIN_ROOT/abcd" ] && exit 0`. Steady-state
   sessions pay one file test and no network (intent AC).
2. **Platform gate.** `uname -s`/`uname -m` normalised to
   `{darwin,linux} × {amd64,arm64}`. Anything else prints the plain-language
   unsupported-platform message (naming the supported matrix and the
   build-from-source path) and exits 0 without changing anything — exit 0
   because an unsupported platform is a reported condition, not a hook error
   to retry every session.
3. **Concurrency lock.** `mkdir "$CLAUDE_PLUGIN_ROOT/.bootstrap.lock"` as the
   mutex (mkdir is atomic on POSIX). A second concurrent session that loses
   the race exits quietly; a stale lock older than 10 minutes is removed and
   retaken. The lock is removed on every exit path (`trap`).
4. **Download.** `curl -fsSL --max-time 120` fetches
   `releases/latest/download/abcd-<os>-<arch>` and `checksums.txt` into a
   temp dir under the plugin root (same filesystem, so the final rename is
   atomic). The latest-release *tag* is read from curl's effective URL
   (`-w '%{url_effective}'` on the redirect); `release_sha` is resolved from
   the GitHub API when reachable and recorded as `unknown` otherwise — the
   meta file never guesses.
5. **Verification.** The `checksums.txt` line for the platform binary is
   checked with `shasum -a 256 -c` (fallback `sha256sum -c`). A mismatch or
   an absent manifest line deletes the temp dir and fails loudly naming the
   mismatch — a corrupted or unpublished artefact never reaches the binary
   path (intent AC). Same-origin checksums are the accepted bar per the
   intent's warrants; the failure message names build-from-source as the
   full-trust alternative.
6. **Install.** `chmod 0755` then `mv` (rename, same filesystem) onto
   `$CLAUDE_PLUGIN_ROOT/abcd`; write `.binary-meta`; suggest — once, in this
   success message — `abcd ahoy install` for terminal PATH setup (the
   PATH story stays owned by ahoy's existing symlink logic, per the intent's
   Decisions).
7. **Failure message.** Every failure path emits one plain-language message:
   what is missing, why it matters (hooks cannot run; shell guard is
   inactive), and the one command that fixes it (retry online / manual
   README install / build from source). No raw "No such file or directory"
   ever reaches the user from this script. Per the intent's Decisions the
   *other* hooks keep failing loudly during the window — that is today's
   behaviour and is deliberately left in place for this slice.

### Hook wiring

`hooks.json` declares ONE `SessionStart` entry, whose command runs the
bootstrap and then the two binary-backed calls in a single shell:

```json
{"type": "command", "timeout": 240, "command": "… bootstrap.sh …; i=$(cat); … abcd hook session-start; … abcd hook prompt-router-reset"}
```

Two properties of that chain are load-bearing and easy to lose. **The payload is
read once and piped to each call separately.** Every hook verb takes its input
with `io.ReadAll` over the whole of stdin (`readHookInput`), so two calls sharing
one stdin leave the second reading EOF — `hook session-start` would fail to
unmarshal and take its silent `return nil` path, disabling both of its notices
in every session. **`session-start` runs before `prompt-router-reset`**, because
the reset ends with an unconditional "abcd rules: reset session" diagnostic on
stderr, and whichever call runs first owns the only line the transcript renders.
The exit precedence (bootstrap, then `session-start`, then the reset) is
computed from saved status codes and does not depend on that order.

**Correction (2026-08-12, iss-204 / iss-208).** This section wired the
bootstrap as the first of THREE sibling `SessionStart` entries on the warrant
that *"ordering within one event's hook list is preserved by the harness, so
the binary-backed session hooks run after the bootstrap in the same event"*.
That warrant is **false**, and it was load-bearing: the harness runs every hook
matching an event **in parallel** (the hooks reference says so verbatim). Both
gated entries raced the ~10.7 MB download, lost, printed "the plugin binary is
not installed", and genuinely did not run — on every fresh install and every
plugin update, since an update lands in a fresh commit-stamped cache directory
with no binary. The three entries are collapsed into the one command above, so
the sequencing is OWNED by the manifest rather than assumed of the harness; the
two gated messages become one; and the bootstrap's own message is emitted
first, so the single line the transcript renders is the success rather than a
missing-binary complaint (iss-208). Nothing else in this spec rests on hook
ordering. Do not restore the ordering warrant.

The `UserPromptSubmit`/`PreToolUse`/`PreCompact`/`SessionEnd` commands are
unchanged — on the first-ever event before any `SessionStart` completed they
fail as today (fail-open guard with UNGUARDED warning, per the intent's
guard-window decision).

### Version-skew notice

At session start, when `.binary-meta` exists and `plugin_sha` (the cache
directory's commit stamp) differs from `release_sha`, the session-start
surface appends one line: surface commit, binary release tag, and that newer
merged fixes may not yet be in the binary. When `release_sha` is `unknown`
no claim is made — visibility never becomes guesswork. The comparison and
rendering live in the existing session-start hook path (surface tier);
`.binary-meta` parsing is a few lines and carries no core semantics.

### Guard health

`GuardHealth.BinaryReachable` already probes `pluginBinaryPath(pluginRoot)`.
The rendered detail for an unreachable binary is extended to name the
bootstrap (`hooks/bootstrap.sh runs at session start; retry online or install
per the README`) so `abcd ahoy` answers "why is the guard down and what do I
do" in one place (intent AC).

## Testing

- **Script tests** (Go, `internal/surface/cli` or a dedicated
  `hooks/bootstrap_test.go` runner): execute `bootstrap.sh` with
  `CLAUDE_PLUGIN_ROOT` pointed at a temp dir and the download URL overridden
  to an `httptest` server (`ABCD_BOOTSTRAP_BASE_URL`, test-only override,
  mirroring `ABCD_BIN_TARGET`'s pattern). Cases, each watched fail first:
  fast path (no server hit when binary exists); happy path installs and is
  executable; checksum mismatch installs nothing and names the mismatch;
  absent manifest line refuses; unsupported platform (overridden uname via
  PATH shim) changes nothing, exits 0, names the matrix; lock contention
  (pre-created lock) exits quietly; stale lock is broken.
- **Guard-health test**: missing plugin-root binary renders the recovery
  detail.
- **Skew test**: meta with differing `plugin_sha`/`release_sha` renders the
  one-line notice; `release_sha=unknown` renders nothing.
- **CI smoke**: the existing smoke job additionally runs
  `sh hooks/bootstrap.sh` against a fixture server on both OS runners —
  the fresh-install AC exercised end to end without touching GitHub.

## Acceptance-criteria mapping

| Intent AC | Delivered by |
| --- | --- |
| Fresh install, online → all hooks work | bootstrap steps 1–6 + wiring |
| Update into fresh cache dir heals | same path; nothing persists but what bootstrap re-creates |
| Checksum mismatch / missing manifest → refuse, name it | step 5 |
| Unavailable binary → actionable message, fail-open guard, ahoy reports | step 7 + guard health |
| Unsupported platform → plain statement, no change | step 2 |
| Surface newer than binary → surfaced | `.binary-meta` + skew notice |
| Present binary → no network, no latency | step 1 |
