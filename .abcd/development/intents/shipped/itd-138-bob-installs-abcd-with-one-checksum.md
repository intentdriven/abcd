---
id: itd-138
slug: bob-installs-abcd-with-one-checksum
spec_id: spc-40
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-135]
severity: minor
impact: additive
---

# Bob installs abcd with one checksum-verified command served from the project's own domain

## Press Release

> **Bob installs abcd with one command served from the project's own
> domain.** Bob rolls tools out to a platform team, and every install path
> they adopt is a trust decision. `curl -fsSL https://abcdev.app/install.sh | sh`
> now does exactly what the README's one-liner does — detect the OS and
> architecture, refuse unsupported platforms in plain language, download the
> binary and `checksums.txt` from GitHub's permanent latest-release redirect,
> verify the SHA-256 and refuse on a mismatch or a manifest that does not
> list the binary, install to `~/.local/bin` without ever escalating
> privileges, print the PATH fix when needed, and finish with `abcd version`
> — because the script and the one-liner are rendered from the same
> template. The whole script runs through `main "$@"`, so a truncated
> download executes nothing. A visible link beside the command reads the
> script before running it, and the by-hand paragraph stays for anyone who
> will not pipe to a shell. That is the entire endpoint: no Homebrew tap, no
> redirect service, no second trust surface. "I read the script, checked the
> checksum step, and rolled it out," said Bob. "The one-liner on their README
> and the script on their domain could not disagree with each other, and
> that is what I am really buying."

## Why This Matters

An install command served from the project's own domain is the norm among
single-binary CLI tools, and its absence reads as immaturity; but every
additional distribution channel is a standing trust surface. Rendering the
script and the README one-liner from one template makes agreement structural
— a test asserts the same URLs and the same checksum step, with only the
platform detection resolved — and GitHub's version-free release asset names
plus the permanent `releases/latest/download/` redirect mean no redirect
infrastructure is needed at all. The deliberate refusals (no Homebrew, no
`/latest` redirects, no attestation page) are recorded with the plan and
stay one-line "later" items rather than silent gaps.

## Acceptance Criteria

- Given `https://abcdev.app/install.sh`, then it is served as a static asset
  with `Content-Type: text/plain; charset=utf-8`, generated from a template
  under `site-src/` that also renders README's one-liner, and a test asserts
  the script, the README form and the per-OS forms in
  `docs/how-to/install.md` share the same URLs and checksum step.
- Given a supported platform, when the script runs, then it downloads the
  platform binary and `checksums.txt` from
  `https://github.com/Partnermedia/abcd/releases/latest/download/`, verifies
  the SHA-256, refuses on a mismatch or a manifest that does not list the
  binary, installs to `~/.local/bin` (overridable with `--bin-dir`), never
  escalates privileges, prints the PATH fix when the directory is not on
  PATH, and ends by running `abcd version`.
- Given an unsupported platform, then the script refuses with a
  plain-language message naming the manual path (the releases page).
- Given a truncated download, then nothing executes, because the script's
  body runs only through a final `main "$@"`.
- Given the landing page's install strip, then a visible read-the-script
  link sits beside the command, and the manual-install paragraph with the
  releases link remains.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-06e2fd44220d -->
Fidelity review — receipt rcp-06e2fd44220d (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:bdf503556685bdbea41a2a2c20652c9158ec59bd50f52d34f8a7b901dcc138f7
Input attestations: diff:site-src, internal/core/site/{build,compose}.go, internal/core/site/installsurface_test.go, README.md and docs/how-to/install.md at chore/audit-run-a-1 80b44890 (git ls-tree -r; spc-40 closed, itd-138 shipped)@sha256:4e73eca06f1233fad80c43dfe9fe52dcba34bf57523b665fe50321363146549e;

Acceptance rollup: MET 3 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: the build renders /install.sh from site-src/install.sh.tmpl, the committed headers serve it as text/plain; charset=utf-8, and TestInstallSurfacesAgree holds README's fenced one-liner, the template and install.md's macOS and Linux forms to one release prefix, the anchored checksums.txt lookup and the same verifier; the concern is that README's one-liner is a hand-written sh -c block the test READS from README.md — the template does not render it, so the agreement the criterion promised structurally is held by assertion instead
  evidence: internal/core/site/build.go:153 — "installTemplateRelPath = "site-src/install.sh.tmpl""
  evidence: site-src/headers:29 — "Content-Type: text/plain; charset=utf-8"
  evidence: internal/core/site/installsurface_test.go:158 — "func TestInstallSurfacesAgree(t *testing.T) {"
  evidence: internal/core/site/installsurface_test.go:308 — "script: onlyInstallBlock(t, "README.md", fencedBlocks(readFile(t, readmePath))),"
  evidence: README.md:106 — "sh -c 'set -eu; unset HTTPS_PROXY"
- ac-2 — MET_WITH_CONCERNS: the script fetches the platform binary and checksums.txt from the latest-release download prefix with a hardened curl, refuses a manifest that does not list the binary and a SHA-256 mismatch, installs to $HOME/.local/bin or --bin-dir, dies rather than escalating when the directory cannot be written, prints the PATH export line when the directory is off PATH and ends by running the installed binary's version verb; the concern is that the prefix is github.com/intentdriven/abcd, not the Partnermedia/abcd the criterion names — the forge handle moved after the promise was written
  evidence: site-src/install.sh.tmpl:87 — "https://github.com/intentdriven/abcd/releases/latest/download/$1"
  evidence: site-src/install.sh.tmpl:98 — "die "checksums.txt does not list $1, so the download cannot be verified. Nothing is installed.""
  evidence: site-src/install.sh.tmpl:100 — "printf '%s\n' "$line" | sha256sum -c - >/dev/null ||"
  evidence: site-src/install.sh.tmpl:115 — "abcd never escalates privileges — pick a directory you own with --bin-dir DIR."
  evidence: site-src/install.sh.tmpl:127 — "is not on your PATH. Add this line to your shell profile"
  evidence: site-src/install.sh.tmpl:130 — ""$1/abcd" version"
  evidence: site-src/install.sh.tmpl:134 — "bin_dir="$HOME/.local/bin""
- ac-3 — MET: an unsupported os-arch pair dies with a plain-language message naming the released platforms, the releases page and the build-from-source path
  evidence: site-src/install.sh.tmpl:76 — "die "there is no released abcd binary for $1-$2."
- ac-4 — MET: the script defines every function first and its last line is the sole call site, so a body cut short in transit defines less and runs nothing; no test asserts the final-line shape, which the gap audit notes
  evidence: site-src/install.sh.tmpl:179 — "main "$@""
  evidence: site-src/install.sh.tmpl:14 — "`main "$@"` runs anything, so a download truncated in transit executes nothing."
- ac-5 — MET: the install chapter writes a read-the-script link to /install.sh beside the command (from the ui.json string) and keeps the small paragraph linking the latest release, checksums.txt, the changelog and all releases under the tabs
  evidence: internal/core/site/compose.go:1055 — "read = `<p class="small muted readscript"><a href="/` + installScriptName + `">` +"
  evidence: site-src/ui.json:102 — ""read_script": "read the script","
  evidence: internal/core/site/compose.go:1075 — "`<a href="` + escapeAttr(rr+"/releases/latest") + `">` + escapeText(c.ui.LatestRelease) + `</a> · ` +"

Gap audit:
- honoured:
  - /install.sh is a text/plain static asset rendered from a site-src template
    evidence: site-src/headers:29 — "Content-Type: text/plain; charset=utf-8"
  - checksum-verified, privilege-free install ending in the version verb
    evidence: site-src/install.sh.tmpl:130 — ""$1/abcd" version"
  - the three surfaces are held to one release prefix and checksum step by a test
    evidence: internal/core/site/installsurface_test.go:158 — "func TestInstallSurfacesAgree(t *testing.T) {"
  - read-the-script link beside the command and the releases paragraph
    evidence: internal/core/site/compose.go:1055 — "read = `<p class="small muted readscript"><a href="/` + installScriptName + `">` +"
- diverged:
  - the template also renders README's one-liner
    evidence: internal/core/site/installsurface_test.go:308 — "script: onlyInstallBlock(t, "README.md", fencedBlocks(readFile(t, readmePath))),"
  - downloads come from github.com/Partnermedia/abcd
    evidence: site-src/install.sh.tmpl:87 — "https://github.com/intentdriven/abcd/releases/latest/download/$1"
- missing:
  - a test that the script's final line is `main "$@"`
    evidence: site-src/install.sh.tmpl:179 — "main "$@""
