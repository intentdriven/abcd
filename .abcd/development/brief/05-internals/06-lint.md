# Lint Contract

Canonical reference for the lint engine in `internal/core/lint` — the deterministic drift and currency gates abcd ships. It describes what the engine covers, the severity model (§2), and how the gates run in CI (§3).

## 1. Lint coverage

The lint engine lives in `internal/core/lint` (Go). It is driven by two armed, deterministic gates, each reading its own JSON rule config as the single source of truth for the armed rule set:

- **Record-currency** (`cmd/record-lint`, config `.abcd/record-lint.json`) lints the markdown design record under `.abcd/development/` for drift: frontmatter and schema shape, resolvable cross-links, directory coverage, intent-lifecycle placement, retired or banned tokens, index-drift on generated regions, delivery-state agreement, citation currency, and record ids cited in record PROSE (`prose_citation_resolves`: an id written in a record's prose must name a record that exists, unless the author marks the line `<!-- record-lint: illustrative -->` or `<!-- record-lint: forward-looking -->`, or the id is carried by the ratcheting baseline `.abcd/prose-citations-baseline.json`; §1.1 says exactly what counts as prose). `make record-lint` runs it, and CI runs the same job on every push.
- **Docs-currency** (`abcd lint docs`, config `.abcd/docs-lint.json`) lints `docs/` and the repo-root prose for change-narration (past-tense drift such as "previously" or "formerly"), broken relative links, stray root markdown, host-name leakage, British-spelling drift, em-dash-in-list-item punctuation, and citation health. `make docs-lint` runs it, and CI runs it on the Linux leg.

Each rule carries a severity (`blocker`, `warn`, or `info`) resolved from its config entry; the severity model is §2. A rule is enabled, disabled, or re-severitied by editing its config entry, so the armed set is always the JSON config, never this document.

The engine carries no numbered code catalogue: rules are named, not numbered, and their definitions live in the two JSON configs above. Any literal enumeration of rules in the record belongs in a generated, gated region (an `index_drift`-style marked block that fails when it drifts from the config), never a hand-kept table — a hand-typed catalogue goes stale against the engine's own rules the moment a rule is added.

### 1.1 What `prose_citation_resolves` reads

The rule reads a record's free text, which is wider than its body and narrower than its bytes. Four boundaries an author meets:

- **Frontmatter free text is prose.** The whole document is read, minus the frontmatter lines whose key is one of the typed cross-reference fields `record_schema` already resolves (and the indented block under such a key). Everything else above the `---` — `deferral_reason`, `found_during`, `resolution`, a `kind_notes` sentence — is a sentence someone wrote, and an id inside one must resolve like any other. A YAML comment after the value carries the line marker where a value must stay verbatim: `found_during: "…" # <!-- record-lint: illustrative -->`.
- **A slug does not stop an id being an id.** `itd-160-dangling-….md` in a sentence cites `itd-160`. `links_resolve` judges markdown link *targets*, `[..](..)`, so a bare filename-shaped handle in prose reaches no other rule; treating the shape as a filename let an invented id go quiet under an appended slug. A placeholder written with a LETTER — `itd-N`, `spc-<id>`, `adr-NNNN` — is still not a citation and still needs no marker.
- **Only triple-backtick fences are code.** A `~~~` fence is not recognised, and neither is four-space indented code: an id inside either is read as prose and must resolve or carry a marker. The rule fails toward asking rather than toward silence, and this is the one place an author meets that.
- **Nine stores are scanned; four families resolve.** The `record_stores` config names nine roots so every record's prose is read, but only `adr`, `itd`, `iss` and `spc` are the cited-id grammar. An `rdi`, `dsp`, `rdg`, `adm` or `srp` id is not a citation to this rule and is checked by nothing here — those stores are in the list for the prose their files carry, not for their own ids.

The committed baseline `.abcd/prose-citations-baseline.json` carries the ids that predate the rule, one entry per id with a class and a note, and it ratchets down: an entry whose id resolves or that nothing cites any more is reported as spent (`prose_citation_baseline_stale`, `info`). An entry is a GLOBAL licence for its id, so a mention that can carry a line marker takes the marker instead.

## 2. Severity model

Three severities, resolved per rule:

| Severity | Meaning | Effect |
|---|---|---|
| `blocker` | The record or docs must not ship with this finding | Fails the gate: non-zero exit, CI fails |
| `warn` | Surfaced for attention, does not by itself mean the record is wrong | Raises the exit code to 1 on an otherwise-clean run (the tri-state grammar in §3) |
| `info` | Advisory only | Never affects the exit code |

Severity is a per-rule field in the owning JSON config (`.abcd/record-lint.json` or `.abcd/docs-lint.json`). Raising or lowering a rule's weight, or disabling it (`enabled: false`), is a config edit, never an engine change; the config is the authority for its gate's armed set and severities.

## 3. CI integration

Two gates are armed today; the finer-grained tiers below them are design targets, not yet built.

**Armed today.** Both gates run on every relevant change:

- **`make record-lint`** runs `cmd/record-lint` over the design record, and CI runs the same job on every push. It fails the build on any `blocker` finding.
- **`make docs-lint`** runs `abcd lint docs` over `docs/` and the repo root, and CI runs it on the Linux leg. It fails on change-narration in a doc body, a broken relative link, a stray root markdown file, or any other `blocker`-severity rule.

Both gates read git-tracked bytes through `internal/core/lint` and honour the severity model in §2. Their exit code follows the standalone tri-state grammar — **blocker → 2, warn → 1, clean → 0** (an error dominates a warning; the tri-state lives in `internal/core/repolint`) — so a CI job that branches on the exit code treats any non-zero value as failure.

**Design targets (not yet built).** These tiers are planned; the record describes them as intent, not as present reality:

- **A staged-tier pre-commit hook** that lints only the files a commit stages and filters findings to the staged lines, so a pre-existing finding on an untouched line never blocks a commit.
- **A changed-lines PR tier** that lints only the lines a pull request changes against its merge base (never a blanket repo scan), so historical files are grandfathered and untouched legacy lines are never retroactively failed.
- **A full-corpus tier** for checks that can only be verified against the whole record (for example a promote back-link between an intent and its source issue, which needs both files in view).

Until those ship, the two armed gates above are the whole of CI's lint coverage: they run over the tracked record and docs on every push, not over a staged or changed-lines slice.

## 4. Output format

Each gate prints human-readable findings by default and machine-readable JSON on `--json`. A finding names its rule id, severity, the repo-relative file, the 1-based line, a message, and (where the rule offers one) a remediation fix. The run result carries the findings, the list of skipped rules, and the tri-state exit code (§3). The human render adds colour on a TTY; the JSON form is the stable contract for tooling that consumes gate output.

## 5. Adding or changing a lint rule

A rule is added, tuned, or retired by editing the JSON config that owns it — `.abcd/record-lint.json` for the record gate, `.abcd/docs-lint.json` for the docs gate — not by editing this document:

1. Add or amend the rule entry in the owning config, with its pattern or check, its `severity`, and (for a banned token) its message and allowed-context escapes.
2. If the rule needs logic the config shapes cannot express, extend the engine in `internal/core/lint` and wire the new check to a config key, so the armed set stays config-declared.
3. Watch the rule fail on a fixture before it passes, so the gate is evidence-backed rather than an enforcement claim.

The `documentation-auditor` agent (per `01-agents.md`) audits this contract on every disembark: a gate the record names must resolve to a live config entry or engine check.

## 6. Acceptance (the lint contract is itself acceptance-checked)

- **Given** the record names a lint gate, rule, or config, **when** the `documentation-auditor` runs at disembark, **then** the auditor fails if the named gate or rule does not resolve to a live entry in `.abcd/record-lint.json` / `.abcd/docs-lint.json` or a check in `internal/core/lint`.
- **Given** a rule's severity is changed in its config, **when** the owning gate runs, **then** the finding is emitted at the configured severity and the exit code follows the tri-state grammar in §3.
- **Given** a planned lint tier that is not yet built (§3), **when** the record describes it, **then** it is written as intent — a design target — rather than in the present tense that would read as shipped.

## References

- `.abcd/record-lint.json`, `.abcd/docs-lint.json`: the armed rule configs — the single source of truth for what each gate checks and at what severity.
- `internal/core/lint`, `internal/core/repolint`: the Go lint engine and its tri-state exit grammar.
- `05-prompt-quality.md`: the prompt-quality discipline this contract sits alongside.
- `01-agents.md`: the `documentation-auditor` that audits this contract.
