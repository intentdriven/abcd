---
id: dcn-2608220000000009
slug: all-dimensions-bug-hunt-round-7-branch
date: 2026-08-22
---
2026-08-22 — All-dimensions bug-hunt round 7 (branch `bughunt-b/round-7`).
Fixed 11 substantive and 4 nitpick findings across five hunt dimensions, each
adversarially refuted before fixing and each behaviour change pinned by a
watched-fail test. Security/code: the guard tokenizer now recognises `$'...'`
ANSI-C and `$"..."` locale quoting (a Tier-1 blocker bypass); the hard-fail
`local_username` redaction matcher folds case; five site-generator fixes
(SVG prolog/comment/CDATA build-vs-check disagreement, unparsed-page gate
masking, unscreened repository `href` scheme, record-id collision, non-ASCII
path dates); the launch scaffold's symlink path leak and its `./internal/...`
race leg that wedged a bare adopter; a capture `skipped[].error` absolute-path
leak. Infra/docs/record: CODEOWNERS now covers `site-src/` and
`docs/requirements.txt`; the shipped `abcd site check` is called shipped in the
brief and pinned in the release-gate manifest; the site-build read set, the
writing-style escape scope, the invariants and evidence-chapter banners, and
several doc/string nitpicks corrected. The guard brace-expansion bypass
(`{--force,}`) is recorded open (iss-2608221457227161) as a distinct
expansion-not-quoting follow-up. Refuted/out of scope this round: the site.md
"runs in CI" claim (in-flight PR #436), the iss-315 guard-warn citation, the
install-surface transport-pin and `-q` deltas (defence-in-depth, no reachable
attack), the record-lint job-name and prepare-this-repo template path (prior
art iss-304/iss-87).
