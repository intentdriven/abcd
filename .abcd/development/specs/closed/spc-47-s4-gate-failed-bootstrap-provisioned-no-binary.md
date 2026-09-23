---
id: spc-47
slug: s4-gate-failed-bootstrap-provisioned-no-binary
intent: itd-154
---
# s4-gate-failed-bootstrap-provisioned-no-binary

## Summary

Makes the plugin's SessionStart bootstrap reliably provision the abcd binary on a
no-Go machine: a visible, loud-staged download of the checksum-verified release
binary into the plugin root, a fail-loud path when the download or checksum fails
(never a limping `No such file or directory` on every hook), and a structural
guarantee that the plugin release payload bundles no platform binaries. The Cut A
§4 gate failed because on a cloud-synced machine bootstrap emitted nothing —
neither success nor refusal — and left the plugin root with a source checkout but
no binary. This spec closes that silent-degradation gap and adds an automated
self-check for it.

## Scope

In:

- `hooks/bootstrap.sh` — the provisioning notice, the terminal success/refuse
  contract, and the guarantee that bootstrap never exits silently on the
  degraded-environment path.
- An automated fresh-install self-check (the §4 assertions as a runnable gate):
  one bootstrap success line, a provisioned plugin-root binary, `/abcd`
  responsive.
- `internal/core/launch/bundle.go` — confirm and test the structural deny that
  keeps platform binaries out of the payload.

Out:

- No move to build-from-Go: the binary arrives only via the checksum-verified
  download (there is deliberately no build-from-source branch).
- No change to the release workflow's binary/checksum production
  (`.github/workflows/release.yml`), which already emits `abcd-*` + `checksums.txt`.

## Approach

**Most of the machinery already exists — the gap is observability and fail-loud
reliability.** `hooks/bootstrap.sh` already downloads the per-platform asset
`abcd-$os-$arch` from the release (`bootstrap.sh:248–267`, URL 440/454),
downloads `checksums.txt` (456), verifies via `shasum -a 256 -c` / `sha256sum -c`
(464–472), and re-verifies on every promotion into the plugin root (586–590). It
emits a staged notice via `notice()` (126) and a success line (685), with a
single loud `refuse()` failure path (98) covering download (455), checksums
(457), missing entry (462) and mismatch (472). The §4 failure was that on a
cloud-synced home (anomalous `stat`, 65535 link counts, multi-minute tree walks)
the chain produced *no* output at all — the loud staging never reached the
transcript.

**Guarantee a terminal line.** Harden `bootstrap.sh` so every exit path emits
exactly one terminal marker — a `provisioning the abcd binary…` line at the start
of the provisioning attempt (AC2) and, at the end, either the existing success
line or a `refuse()` with a clear message (AC4). The provisioning notice is moved
to fire *before* the environment-sensitive work (the tree walk / stat) so a hang
or failure there still leaves the "provisioning…" marker in the transcript, and a
trap on abnormal exit converts a silent death into a `refuse()`. The result is
that a degraded environment produces a loud refusal, not a silent no-op that
leaves `UserPromptSubmit`/`PreToolUse` erroring `No such file or directory`.

**No bundled binaries — structurally.** The launch bundler
`internal/core/launch/bundle.go` is a default-deny taxonomy; platform binaries
are not committed and never enter the payload (they exist only as release
assets). This spec adds a payload-completeness assertion that no `abcd-darwin*`/
`abcd-linux*` artefact is ever includable, pinning AC5 structurally.

**The self-check gate.** Add a runnable check that encodes the §4 assertions so
the failure family is caught automatically rather than by a manual evening: on a
fresh plugin install it asserts (a) exactly one bootstrap success line appeared,
(b) a provisioned binary exists at the plugin root (not merely a source
checkout), and (c) `/abcd` answers in about a second with no Go toolchain
present. This is the detector the intent names ("the §4 checklist") turned into a
gate.

## How it satisfies each acceptance criterion

- *No-Go machine, plugin installed, session starts → bootstrap downloads the
  platform binary and `/abcd` answers in about a second* — the existing download
  path (bootstrap.sh:248–267, 440–472) with the reliability hardening; the
  self-check asserts `/abcd` responds. Test: the fresh-install self-check on a
  Go-absent environment.
- *A visible loud-staged `provisioning the abcd binary…` line, never silent* —
  the notice is emitted before the environment-sensitive work and guaranteed on
  entry to provisioning. Test: assert the marker appears in the bootstrap output
  even when the subsequent step is made to fail.
- *Downloaded binary verified against the release checksum before use* — the
  existing `checksums.txt` download + `shasum -c` gate (456–472), re-verified on
  promotion (586–590). Test: a tampered artefact fails the checksum step.
- *Download or checksum failure fails loudly with a clear message, not a limping
  hook state* — the `refuse()` paths (455/457/462/472) plus the abnormal-exit
  trap that converts a silent death into a refusal. Test: simulate a download
  failure and a checksum mismatch; assert a loud refuse and that no partial
  binary is left that would half-satisfy the hooks.
- *The plugin release payload bundles no platform binaries* — the structural deny
  in `bundle.go`. Test: a payload-completeness assertion that `abcd-*` binaries
  are never includable.

## Decisions

Download-only, no build-from-source: the no-Go promise is the whole point, so the
binary must arrive by checksum-verified download, and `refuse()` only *names*
`go build` as a manual escape hatch rather than a bootstrap branch. The fix for
the §4 failure is observability and fail-loud, not a new download mechanism — the
download already worked; what failed was that a degraded environment produced
silence. Guaranteeing a terminal success-or-refuse line, and gating it with an
automated self-check, is what makes the no-Go promise provable rather than
hoped-for.
