---
id: spc-35
slug: hook-binary-to-persistent-data-dir
intent: itd-132
---
# hook-binary-to-persistent-data-dir

## Summary

spc-35 delivers itd-132's maintainer-ruled shape: the harness's persistent
per-plugin data directory (`${CLAUDE_PLUGIN_DATA}`) becomes a **download
cache** for the checksum-verified release artefact, plugin roots are
provisioned from it by verified copy, and the PATH entry becomes an
abcd-owned **regular file** instead of a symlink into a harness-owned
directory. Execution stays per-root — the relocate-execution alternative was
rejected in adversarial review for creating a shared mutable binary across
roots. Supersedes spc-21's "update heals by re-fetch" criterion (replacement
contract: an update never re-fetches unless the released binary changed);
every other spc-21 invariant — same-origin checksums, HTTPS pin, `-q`-first
curl, proxy/CA unset, no environment seam for origins, refuse/notice
message discipline — carries over verbatim.

## Design

### 1. Cache layout (in `${CLAUDE_PLUGIN_DATA}`)

```
${CLAUDE_PLUGIN_DATA}/
  cache/abcd-<os>-<arch>     the verified release artefact (0755)
  cache/binary-meta          release_tag / release_sha / binary_sha256 /
                             fetched_at — the spc-21 format minus plugin_sha
  path-entry                 provenance of the abcd-owned PATH copy:
                             installed path + binary_sha256
  .bootstrap.lock            relocated concurrency lock (mkdir-atomic)
  .bootstrap.tmp.$$          relocated temp dirs (same-filesystem rename)
```

`plugin_sha` is dropped from the meta: with one cache serving many roots, a
recorded provisioning-time root is meaningless — the version-skew notice is
computed against the **live** `CLAUDE_PLUGIN_ROOT` basename at render time
(40-hex gate unchanged).

### 2. Provisioning flow (bootstrap.sh rework)

Steady state first, as in spc-21: a regular executable at
`$CLAUDE_PLUGIN_ROOT/abcd` exits in one file test, no network (AC 3).

On a missing root binary (fresh install or fresh post-update root):

1. If `CLAUDE_PLUGIN_DATA` is unset or unusable: degrade **loudly** to the
   current spc-21 per-root fetch — the loud-staging principle; never a
   silent fallback (AC 7). Deriving the documented path shape from
   `CLAUDE_PLUGIN_ROOT` is deliberately not attempted: the derivation is
   documented but not endorsed, and a wrong guess plants a trusted artefact
   in an untracked location.
2. Take the relocated lock in the data dir (per-root locks cannot serialise
   two roots writing one cache). The per-root `.bootstrap.attempt` throttle
   in the hook commands stays as-is — it only rate-limits invocation.
3. **Refresh detector** (maintainer-ruled, local state + one best-effort
   resolve): re-resolve the latest release tag exactly as spc-21 step 4.
   - Resolve answers a tag **equal** to `cache/binary-meta`'s `release_tag`:
     the cache is a candidate, but its co-located record does not by itself
     establish trust (adr-46 decision 3 — a same-UID attacker writes both the
     artefact and the record that vouches for it). **Authenticate the cache
     against the published manifest**: fetch `checksums.txt` for the resolved
     release (same-origin, HTTPS-pinned, `-q`-first — the exact spc-21 step 5
     manifest fetch), and check the cached `binary_sha256` against the
     published digest for `abcd-<os>-<arch>`. On match, provision from cache —
     re-hash `cache/abcd-<os>-<arch>` (catching corruption in the file itself),
     copy into a data-dir temp dir, rename-install into the plugin root (AC 1,
     AC 5). On mismatch, the cache is tampered or stale: discard it and fall to
     the different-tag download path. If the manifest fetch fails but the tag
     resolved, treat as offline (below).
   - Resolve **fails** (offline): no published manifest is reachable, so the
     cache is trusted at **corruption-evidence only** — re-hash the artefact
     against its recorded `binary_sha256` and provision on match; a mismatch
     refuses loudly and installs nothing. This is the deliberate availability
     trade (adr-46 decision 3), and the success notice says so: provisioned
     from an unauthenticated cache while offline, not "verified against the
     published manifest".
   - Resolve answers a **different** tag (or the equal-tag manifest check
     rejected the cache): download + verify into the cache under the unchanged
     spc-21 steps 5–6, update `cache/binary-meta` by rename, then provision the
     root from the fresh cache (AC 2).
   - No cache and resolve fails: refuse with the spc-21 no-network message.
   The accepted gap: a release cut with no plugin update never triggers —
   the skew notice surfaces it and `abcd update` (itd-130) is the explicit
   refresh path.
4. The hook manifest (`hooks/hooks.json`) keeps invoking `$r/abcd`; only
   bootstrap.sh learns the cache. SessionEnd never reaches this path at all
   (iss-2608210934566223, shipped separately).

### 3. PATH entry: owned regular file (ahoy rework)

- `installPinnedSymlink` becomes `installOwnedCopy`: re-verify the cache
  artefact against `binary_sha256`, copy to the target (default
  `~/.local/bin/abcd`) as a regular file 0755, record path + hash in the
  provenance record (AC 4, AC 5).
- **Ownership is recorded provenance, never content-guessing**: a regular
  file at the target matching the record's hash is owned (idempotent /
  refreshable); anything else classifies foreign and is refused exactly as
  today. A legacy abcd-owned *symlink* (target inside a plugin root or
  cache dir — today's `classifyBinTarget` geometry) is a heal-able gap:
  replaced by the owned copy on `ahoy install` (AC 4).
- **The provenance record is reachable without the harness environment.**
  `ahoy install`, `ahoy uninstall`, and `abcd update` run from a terminal,
  where `CLAUDE_PLUGIN_DATA` is *not* exported — so the record cannot live in
  the data dir behind that variable, or ownership is unprovable exactly where
  those verbs run: uninstall would leave the binary, `ahoy` would report
  abcd's own binary as foreign, and `abcd update` would desync the record it
  just wrote. The record lives in a home-scoped, abcd-owned location derivable
  from the installed copy without any harness variable (the ahoy user-scope
  store, alongside the history store, is the natural home); the data dir stays
  the *cache's* home only. Any read of the record that finds the recorded path
  absent still ownership-checks before acting — the absent-path branch never
  short-circuits to "owned" (closes iss-2608210934566228's sibling comment
  overclaim).
- **The record carries the plugin root, so the CLI can resolve it without the
  symlink.** The old symlink doubled as `resolvePluginRoot`'s route home (its
  target sat inside the plugin root); a regular-file copy severs that, leaving
  every plugin-root verb a no-op from a terminal (iss-2608210934566230).
  `installOwnedEntry` records `plugin_root=` beside `path=`/`binary_sha256=`,
  and `resolvePluginRoot` gains it as a candidate after the env-var and
  executable-ancestor routes. Because a plugin root is a commit-stamped cache
  dir the harness replaces on update, the hook-side bootstrap — which runs on
  the fresh root with `CLAUDE_PLUGIN_ROOT` set — rewrites the recorded
  `plugin_root=` each time it provisions, so the terminal read tracks the
  latest root within one hook firing of any update. `store_symlink_test.go` is
  extended with the copy layout so the iss-170 contract is pinned by behaviour,
  not by the symlink mechanism.
- Refresh: when bootstrap installs a new release into the cache and the record
  exists, it refreshes the PATH copy in the same run (verified copy + rename);
  `ahoy install` heals it on demand. `ahoy uninstall` removes the owned copy
  and the record; the cache itself is left to the harness's
  uninstall-from-all-scopes deletion (open question 2 resolved: abcd removes
  what abcd owns, the harness removes what the harness owns).
- The `--dev` shim is untouched: it builds from the source checkout and
  never writes into the cache or over a verified artefact.

### 4. Migration (one-way, idempotent)

On the first bootstrap run with an empty cache and a verified binary already
in the plugin root: verify it against the root's existing `.binary-meta`
`binary_sha256`; on match, seed the cache (copy + meta rewrite minus
`plugin_sha`); on mismatch or missing meta, ignore it and fetch fresh (AC
5). Root-local `.binary-meta` files stop being written; the skew notice
reads the cache meta.

The empty-cache test must use the same non-regular-file refusal the main
install path uses: **`[ -e "$cache_binary" ] && [ ! -f "$cache_binary" ]` →
refuse**, matching the plugin-root guard already at the main install site. The
`[ ! -f "$cache_binary" ] || exit 0` fast-path form is a directory-shaped
hazard (iss-2608210934566229): a directory planted at `$cache_binary` is not a
regular file, so that test passes, `mv -f` moves the seed *into* the directory,
and a lying `binary-meta` vouches for a directory — every later root then
downloads and refuses, running UNGUARDED across updates. The same `mv`-onto-
directory guard the binary install uses (`if [ -e … ] && [ ! -f … ]; then
refuse`) applies to the cache path.

## Acceptance-criteria mapping

| itd-132 AC | Delivered by |
| --- | --- |
| Update → verified copy from cache, no fetch | Design 2 step 3 (equal-tag / offline branch) |
| New release + update → fetch-verify into cache | Design 2 step 3 (different-tag branch) |
| Steady state: one file test, no network | Design 2 fast path (spc-21 step 1 unchanged) |
| PATH regular file survives updates & GC; legacy symlink healed | Design 3 |
| Every promotion re-verifies binary_sha256; loud refusal | Design 2 step 3, Design 3, Design 4 |
| Skew notice vs live plugin root | Design 1 (plugin_sha dropped; render-time comparison) |
| No data dir → loud per-root fallback | Design 2 step 1 |

## Testing

Each watched fail first, per the spc-21 script-test pattern (fixture server,
temp `CLAUDE_PLUGIN_ROOT` **and** temp `CLAUDE_PLUGIN_DATA`):

- cache hit, resolve offline → root provisioned by copy, zero artefact
  downloads (fixture server records no asset request); success notice names
  the offline/unauthenticated trust.
- cache hit, resolve online → the published `checksums.txt` is fetched and the
  cached hash checked against it before the cache is trusted (fixture server
  records the manifest request, not an asset request); success notice names
  the manifest-verified trust.
- **cache tamper (the case the old test missed): artefact AND `binary-meta`
  rewritten together to a self-consistent poisoned pair, resolve online →
  the published-manifest check rejects the cache and the different-tag
  download path replaces it; nothing poisoned is ever installed.** Watched to
  fail against the co-located-record-only design.
- second plugin root appears (update simulation) → provisioned from cache.
- resolve answers a new tag → one download, cache meta updated, root
  provisioned from the new artefact.
- corrupted cache artefact (bytes only, meta intact) → loud refusal, nothing
  installed anywhere.
- `CLAUDE_PLUGIN_DATA` unset → per-root fetch with the loud degradation
  notice.
- migration: verified root binary seeds the cache once; second run no-ops;
  hash-mismatched root binary is ignored and fetched fresh; **a directory
  planted at `$cache_binary` is refused, not seeded into.**
- ahoy: owned-copy install; original plugin root deleted → PATH entry still
  executes; legacy owned symlink healed to a copy; foreign regular file
  refused; uninstall removes copy + record, leaves the cache.
- **terminal environment (the regression the harness hid): install the owned
  copy with `CLAUDE_PLUGIN_DATA` set, then CLEAR it and every harness var, and
  point the executable at the copy → `Detect` still reports `pinned` and
  resolves the plugin root from the record, `abcd ahoy` does not report abcd's
  own binary as foreign, and `Uninstall` still removes the copy.**
  `store_symlink_test.go` extended with the copy layout.
- skew notice: rendered against a live root differing from `release_sha`;
  silent when `release_sha` unknown (unchanged).

## Documentation surfaces

The PATH entry is now an owned regular file, not a symlink; the plugin
markdown surface and the CLI reference must say so. Update
`commands/ahoy.md` ("pinned … symlink", "switches back to the pinned
symlink", "Remove the marker block and owned PATH symlink") and
`docs/reference/cli/commands.md` to describe the owned copy and its
provenance record.

## Filed alongside (SPLIT routing)

- ADR + brief invariant from iss-2608210934566226 (adr-46, invariant 12):
  persistence must not weaken the spc-21 verification posture at fetch time
  or at rest — every promotion re-verifies, the cache is authenticated
  against the published manifest when online, and SessionEnd performs no
  network work. Invariant 12's SessionEnd clause landed with
  iss-2608210934566223 and is asserted in the register.
- Principle from iss-2608210934566227: store durable state where the
  platform documents it survives.
- CHANGELOG entry at ship; iss-2608210934566221/222 resolved by the
  shipping change.

## Review-driven revisions (2026-08-21)

Design 2 step 3 (cache authentication), Design 3 (terminal-reachable
provenance + recorded plugin root), and Design 4 (migration directory guard)
were revised after the two adversarial reviews of the first implementation;
the surviving findings are iss-2608210934566228 (cache re-verify was
corruption-only), iss-2608210934566229 (migration `mv`-onto-directory), and
iss-2608210934566230 (owned copy broke terminal root resolution). adr-46 is
amended to match.
