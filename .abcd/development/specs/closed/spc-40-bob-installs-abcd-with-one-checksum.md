---
id: spc-40
slug: bob-installs-abcd-with-one-checksum
intent: itd-138
---
# bob-installs-abcd-with-one-checksum-verified-command-served

## Summary

spc-40 delivers itd-138's one new distribution endpoint:
`https://abcdev.app/install.sh`, generated from a template under
`site-src/` that also renders README's universal one-liner, served as a
static asset with the plain-text content type, and held in agreement with
the README form and install.md's per-OS forms by one test. On the itd-140
boundary this is **specific-side**: an opt-in surface of this repository's
composition, not of the record format.

## Settled constraints

- **One template, three surfaces** (adr-47 consequence; itd-138 AC 1): the
  script, README's one-liner and install.md's per-OS forms share the same
  release URLs, the same checksum step and the same install path; a test
  asserts it so a change in one fails the build unless all move.
- **No second trust surface**: no Homebrew tap, no redirect service, no
  attestation page — recorded refusals, not silent gaps.
- **Never escalates privileges**; installs to `~/.local/bin`
  (`--bin-dir` overrides); GitHub's permanent `releases/latest/download/`
  redirect is the only remote.

## Mechanism

- `site-src/install.sh.tmpl` (landed with spc-37's migration) is the one
  source: OS/arch detection, plain-language refusal of unsupported
  platforms naming the releases page, download of the platform binary and
  `checksums.txt` from
  `https://github.com/Partnermedia/abcd/releases/latest/download/`,
  SHA-256 verification refusing on a mismatch or a manifest that does not
  list the binary, `install -m 0755` into the target directory, the PATH
  hint when the directory is not on PATH, and a closing `abcd version`.
  The whole body executes only through a final `main "$@"`, so a
  truncated download runs nothing. Transport hardening:
  `curl --proto '=https' --tlsv1.2 -fsSL`.
- `abcd site build` renders the template to `site/install.sh` unchanged
  (the template is complete shell; rendering is a copy plus a build-stamp
  comment carrying tag and commit), and `_headers` serves it with
  `Content-Type: text/plain; charset=utf-8`.
- The agreement test (Go, in the site package) parses the three committed
  surfaces — README's fenced one-liner, the template, install.md's macOS
  and Linux H3 blocks — normalises the resolved OS detection
  (`uname`/`shasum`/`sha256sum`), and asserts URL set, checksum step and
  install path are identical.
- The landing page's install strip (spc-37's `install` layout) shows the
  command with a visible read-the-script link beside it and keeps the
  by-hand paragraph with the releases link; `docs/how-to/install.md`'s
  lead sentence is updated in the same change that first serves the
  script, so the site never previews a command the repository does not
  ship.

## Acceptance-criteria mapping

- AC 1 (served as static plain-text, generated from the shared template,
  agreement test over the three forms) → template + `_headers` +
  agreement test.
- AC 2 (supported-platform behaviour end to end) → the template's script
  body.
- AC 3 (unsupported platform refuses in plain language naming the manual
  path) → refusal branch.
- AC 4 (truncated download executes nothing) → `main "$@"` structure.
- AC 5 (install strip: read-the-script link and the manual paragraph
  remain) → landing install strip.

## Out of scope

- Homebrew/Scoop/winget/Nix channels, `/latest/<os>-<arch>` redirects, an
  attestation-verification docs section (deliberate refusals, listed as
  one-line later items with the plan).
- A Windows build (install.md carries the one-sentence status; the tab
  follows the page when it changes).
