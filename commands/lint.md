---
name: lint
description: "Check this repository against the working conventions: Writes nothing; refuses with exit 2 on an error finding and exit 1 on warnings alone."
argument-hint: ""
block: people
---

# `/abcd:lint` repo-conformance check

Run the abcd binary's read-only conformance lint for the current repo and
present the result. This command performs **zero writes** — it reports gaps, it
never fixes them (remediation stays with `/abcd:prepare-this-repo`).

Run:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" lint --json
```

Then summarise the JSON for the user. Its shape is `{ "findings": [ … ],
"skipped": [ … ] }`:

- `findings` — each has a stable `ruleId`, a `severity` (`error` or `warn`), a
  `file`, a `line` on content-scanning findings only (`docs-currency`,
  `privacy-hygiene`, `identity-positioning` — path-presence findings omit the
  key entirely), a `message`, and a `fix`. Group them by severity: report
  `error` findings first (these fail conformance), then `warn` findings
  (advisory). For each, give the `file` (with `:line` when present), the
  `message`, and the `fix`.
- `skipped` — rule ids that did not apply to this repo (e.g. `docs-currency`
  when there is no `docs/`). Mention them as "not applicable", not as failures.

State the outcome plainly: if there are no findings the repo conforms; otherwise
lead with how many errors and warnings there are. The process exit code is the
Conftest tri-state — `0` clean, `1` warnings only, `2` any error — so
`abcd lint` can also gate a repo's CI.

## `lint outbound` — judge one piece of outbound text

The sub-verb judges a single artefact rather than the repo: a commit message, a
pull-request body, an issue, a comment, a release note. It applies abcd's
outbound policy — never a live agent-session URL, never a tool's own attribution
footer — and it **reports and refuses; it never rewrites the text**, because the
text belongs to whoever wrote it.

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" lint outbound --label pr-body ./body.md
```

It reads the file named as the positional, or standard input when there is none
(or when it is `-`). `--label` names the artefact in the report; `--root` picks
the repo whose `.abcd/config/pii.json` configures the scan (default: the current
directory).

The exit code is the verdict, and it is **not** the parent verb's Conftest
tri-state: `0` the artefact is clean, `1` the artefact is refused, `2` the check
could not run (an unreadable or empty artefact, a degraded scanner config). Both
patterns are hard-fail, so the tri-state's advisory middle rung has no meaning
here; a caller that branches on non-zero is right either way, and one that
distinguishes must not read "the gate was broken" as a verdict on the text.

With `--json` it emits one document: `label`, `findings` (each with a `kind` of
`harness:session_url` or `harness:attribution_footer`, a `line`, a `column` and a
`suggested_fix`), and the `policy` text. The matched span is masked in both
renderings — a CI log on a public repository is public text, so the gate must not
republish the leak it is reporting. A refusal arrives as the exit status alone;
there is no second error envelope on top of the report.

This is the check abcd's own CI runs over every commit message in a pull
request's range and over the pull-request body
(`scripts/check-attribution.sh`).

A `privacy-hygiene` finding on a deliberately illustrative line can be waived by
adding `abcd-lint:allow` on that line (the earlier `abcd-audit:allow` spelling is
honoured too). No other rule honours that marker: a `docs-currency` finding takes
the docs-lint engine's own `<!-- docs-lint: allow -->` escape, and the remaining
rules have no line waiver — resolve what they report.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
