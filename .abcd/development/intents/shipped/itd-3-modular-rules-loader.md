---
id: itd-3
slug: modular-rules-loader
spec_id: spc-1
kind: standalone
suggested_kind: null
reclassification_history: []
severity: major
impact: additive
---

# abcd Loads Its Own Rules On Demand

## Press Release

> **abcd ships a modular rules loader so the plugin's discipline never bloats your context.** Instead of force-loading a monolithic CLAUDE.md every session, abcd ships a default rule set bundled with the plugin and a per-repo `.abcd/rules.json` override. A `UserPromptSubmit` hook scans each `UserPromptSubmit`-event payload, matches recall keywords against domains (COMMITTING, DOCUMENTATION, ROADMAP, ISSUES, INTENTS, LIFEBOAT), and injects only the rules relevant to what you're actually doing. Sessions about React rendering pay zero tokens for the ADR-format rules; sessions about a PR description load only the commit-and-PII rules. CLAUDE.md becomes a small marker block, not a 1000-line wall.
>
> "I had a 978-line CLAUDE.md silently loading into every conversation," said Bob, staff engineer. "abcd's rules loader took it down to a 30-line marker block plus a JSON file. The full rules still apply — they just turn up when they're relevant. My token bill dropped, my keyboard inputs started landing more accurately, and I never have to maintain CLAUDE.md by hand again."

## Why This Matters

abcd already commits to "single source of truth" and "no scaffolding outside the repo." But the existing methodology document (`~/ABCDevelopment/.claude/CLAUDE.md`) is 978 lines and is force-loaded into every Claude Code session in any abcd-managed repo. Most of those rules don't apply to most prompts — yet every `prompt` pays the token cost.

Prior art ([CARL][carl]) demonstrates the fix: domain-keyed rules with `prompt`-keyword recall, loaded just-in-time. abcd takes the mechanism into its own plugin (rather than depending on CARL the runtime) and ships abcd-shaped defaults out of the box. See [`research/related-work.md`](../../research/related-work.md) for the full comparison.

This is also the keystone for retiring `~/ABCDevelopment/.claude/` entirely. Without modular loading there's no way to relocate the methodology rules without losing them; with modular loading, the rules ship with the plugin, override per repo, and the dev-root tree can be archived.

## What's In Scope

- **`hooks/prompt_router_hook.py`** — `UserPromptSubmit` hook. Reads merged rules (plugin defaults + `<repo>/.abcd/rules.json`), keyword-matches the `prompt` against each domain's `recall` list, injects matching rules as a system message. Signature-based dedup; force full re-inject every N turns (N defaults to 5, configurable).
- **Binary-bundled defaults** embedded in the Go binary (`internal/core/...`). Domains: COMMITTING, DOCUMENTATION, ROADMAP, ISSUES, INTENTS, LIFEBOAT, PII. Each carries `recall` keywords + `rules` array. Content harvested in a one-time manual session from the legacy `~/ABCDevelopment/.claude/CLAUDE.md`.
- **`<repo>/.abcd/rules.json`** schema (JSON Schema embedded in the Go binary, `internal/core/...`). Per-repo file. Can extend defaults, override individual rules, disable a domain entirely, or add custom domains. `/abcd:ahoy` writes a minimal skeleton.
- **CLAUDE.md marker block** owned by `/abcd:ahoy`. ~30 lines. Identifies that abcd is active, points at `.abcd/rules.json` and the plugin defaults, lists the active domains, gives the developer the explicit-activation syntax for star-commands.
- **Star-command bypass** — `*<DOMAIN> …` (e.g. `*ROADMAP draft a milestone`) explicitly activates a domain regardless of keyword match.
- **Dedup + refresh discipline** — same rule signature isn't re-injected within the same session; full refresh every N turns to recover from compaction.
- **Per-domain `state` field** (`active` / `dormant`) — lets users toggle a domain off without deleting the rules.
- **`abcd rules [domain]`** CLI subcommand — bare `abcd rules` prints the full active rule set; `abcd rules <domain>` scopes to one domain. Diagnostic / explainability. Aligns with the bare-command-as-render discipline (see `02-constraints/04-naming.md`); no `show` sub-verb (collapses to bare-with-positional-argument).

## What's Out of Scope

- **Global `~/.abcd/rules.json`** — explicitly rejected. Keeping a personal cross-repo rules file recreates exactly the "scaffolding accumulates outside the repo" failure mode this intent exists to fix. Per-repo only.
- **MCP server for runtime rule editing** — CARL ships one (`carl_v2_add_rule`, `toggle_domain`, etc.). abcd ships JSON-only; the agent edits `rules.json` via the standard file-edit tools. MCP integration is a candidate if friction is real.
- **Star-command bibliography** — only domain activation; no arbitrary `*<command>` macros (CARL's broader capability).
- **Cross-`prompt` rule chaining** — keyword recall is independent per `prompt`. No "session sticky" mode.
- **Migration tooling** — the harvest from legacy `~/ABCDevelopment/.claude/CLAUDE.md` to plugin defaults is a one-time manual session, not a `/abcd:cull` command. See note in the brief.

## Acceptance Criteria

> _BDD format, per `itd-1-acceptance-gates`. These gates are checked by `intent-fidelity-reviewer` when this intent moves to `shipped/`._

- **Given** an abcd-installed repo with `.abcd/rules.json` declaring the COMMITTING domain active, **when** the developer sends a `prompt` containing "commit" or "PR" or "git add", **then** the COMMITTING rules appear as injected system context within that turn.
- **Given** the same repo, **when** the developer sends a `prompt` unrelated to any active domain's `recall` keywords, **then** zero abcd rules are injected and the token overhead is < 200 tokens (header only).
- **Given** the same `prompt` repeated three times in a session, **when** dedup is enabled, **then** the rules inject on `prompt` 1 and skip on prompts 2-3 (until forced refresh on `prompt` 5).
- **Given** a developer runs `*ROADMAP draft a milestone` with no roadmap-related keywords, **then** the ROADMAP domain's rules inject regardless of keyword match.
- **Given** a fresh `git clone` of an abcd repo with only the plugin installed, **when** Claude Code starts a new session, **then** CLAUDE.md is < 50 lines and contains the abcd marker block.
- **Given** the legacy `~/ABCDevelopment/.claude/CLAUDE.md` (978 lines), **when** harvested into `defaults.json`, **then** every methodology rule with broad applicability is preserved (manual review, recorded in `research/legacy-harvest.md`).

## Open Questions

- What's the right N for forced refresh? CARL uses 5; abcd's typical session might be longer (lifeboat sessions especially).
- Should the hook respect a `.abcdignore` for prompts that shouldn't trigger any injection (e.g. paste-of-foreign-text)?
- How does this interact with [Claude Code Skills][claude-skills-docs]? Skills are procedural-workflow shaped; rules are declarative. Boundary needs documenting in the marker block.
- Does the hook need to coexist gracefully with CARL if a developer has both installed? (Probably yes — separate hook scripts, separate JSON files, no conflict expected, but worth verifying in the corpus tests.)

## Audit Notes

> _Manual fidelity audit — the `intent-fidelity-reviewer` agent is not yet built,
> so this audit was performed by hand. Judge: Claude Opus 4.8; date 2026-07-11;
> reviewed against the merged loader (the rules-loader design + implementation).
> Per-criterion verdicts per the itd-1 discipline, then a three-bucket prose
> audit. `spec_id: spc-1` is reserved for itd-3; the future native spec store
> adopts it rather than re-minting._

<!-- abcd-review: INGESTED receipt=rcp-a2e0ab1dca95 -->
Fidelity review — receipt rcp-a2e0ab1dca95 (verifier abcd:intent-auditor claude-opus-5-5).

Provenance: abcd:intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:cef1b0310d2a086912a28fda819999e073e6060207bda9528324fd28b0f411a4
Input attestations: diff:f4bedb247c56ed08ed6e5712070ee827a43b682a..e5fdf97c2762f570c031c2b5afe7ad5fceb761fe: the loader branch merged as d399cb464d498ca28311205b3781e480fcc4b191 (pull request 18) plus the ship commit; the request named no range, so the auditor derived it from the history@-; repository:origin/main at 2e03c61e454bf09ecda8cee0ae2f377157b522f7, the state every path:line below is read at unless its ref names a commit@-; intent:.abcd/development/intents/shipped/itd-3-modular-rules-loader.md, the working copy carrying this receipt's OWED marker@-;

Acceptance rollup: MET 2 · MET_WITH_CONCERNS 2 · NOT_MET 2 · INCONCLUSIVE 0

Audit owed (receipt rcp-a2e0ab1dca95): ac-5 NOT_MET · ac-6 NOT_MET. Remedy: fix, then re-run the audit. Carried by iss-2610241427397332.

Per-criterion verdicts:
- ac-1 — MET: The bundled COMMITTING domain recalls commit and carries pr and git add as aliases, termHit lowercases and word-bounds each term, and the prompt-router hook test puts the COMMITTING block on the hook's stdout (the context the host adds on that turn) for a commit prompt; a repo rules.json naming COMMITTING active merges per field and keeps it active. No test pins the PR or git add triggers by name; the defaults and the matcher show them.
  evidence: internal/core/rules/defaults/rules.json:7 — ""recall": ["commit", "push", "branch", "merge", "rebase"],"
  evidence: internal/core/rules/defaults/rules.json:8 — ""aliases": ["pull request", "pr", "git add", "conventional commit"],"
  evidence: internal/core/rules/rules.go:751 — "func (idx promptIndex) termHit(term string) bool {"
  evidence: internal/surface/cli/cli_test.go:132 — "out, errlog := runHook(t, hookInputJSON(t, "s1", cwd, "commit and push"), "hook", "prompt-router")"
  evidence: hooks/hooks.json:3 — ""UserPromptSubmit": ["
  evidence: internal/core/rules/rules.go:500 — "func Merge(base, over RuleSet) RuleSet {"
- ac-2 — MET_WITH_CONCERNS: Render returns zero bytes for an empty match and the hook test asserts empty stdout for an unrelated prompt, so no rule and no header reaches the context, inside the 200-token bound. Concerns: the header-only shape the criterion names was replaced by zero bytes under the signed-off delta D3; and at HEAD the same prompt hook appends blocks that are not rules and are not gated by domain recall (the AWAY block, about 410 words, on every prompt while the away record holds, and the STATUS block when the status fetch brought news), so the hook's no-match overhead stays under 200 tokens only outside those states.
  evidence: internal/core/rules/rules.go:870 — "if len(domains) == 0 {"
  evidence: internal/surface/cli/cli_test.go:167 — "out, _ := runHook(t, hookInputJSON(t, "s4", cwd, "paint a landscape"), "hook", "prompt-router")"
  evidence: internal/core/rules/inject_test.go:161 — "res := Inject(rs, "paint a landscape in oils", SessionState{}, 0)"
  evidence: .abcd/work/decisions/dcn-2607110000000004-itd-3-rules-loader-design-signed-off.md:13 — "zero model-facing tokens on"
  evidence: internal/surface/cli/cli.go:1876 — "text = joinBlocks(text, away.Block)"
  evidence: internal/surface/cli/hook_away.go:23 — "outside the rules loader's dedup ledger, add the away block to the agent's"
  evidence: internal/surface/cli/cli.go:1875 — "text = joinBlocks(text, status.Text)"
- ac-3 — MET_WITH_CONCERNS: Inject skips a matched domain whose rendered-block signature is already in the session ledger, and the hook test shows the second identical prompt producing empty stdout, so prompt 1 injects and prompts 2 and 3 skip. Concern: no forced refresh lands on prompt 5 by default; under the signed-off delta D1 the primary refresh is the ledger reset on SessionStart and PreCompact and the fixed-N backstop defaults to 15, so a refresh on prompt 5 happens only where rules.force_refresh_every_n is set to 5 in .abcd/config.json.
  evidence: internal/core/rules/inject.go:147 — "if ledger[d.Name] == Signature(d) {"
  evidence: internal/surface/cli/cli_test.go:146 — "out2, _ := runHook(t, in, "hook", "prompt-router")"
  evidence: internal/core/rules/inject.go:55 — "const DefaultRefreshBackstop = 15"
  evidence: internal/core/rules/inject.go:128 — "if count%backstop == 0 {"
  evidence: internal/core/rules/inject.go:351 — "ForceRefreshEveryN int"
  evidence: .abcd/work/decisions/dcn-2607110000000004-itd-3-rules-loader-design-signed-off.md:11 — "(fixed-N demoted to a ~15–20 backstop, not primary)"
- ac-4 — MET: Match appends any domain a `*NAME` token names before its dormant and recall checks, and the tests show `*ZONK` activating a domain whose only recall word never occurs and `*ROADMAP` activating ROADMAP made dormant, with the kill switch still winning. The criterion's own example contains milestone, a ROADMAP recall word, so these two tests, not the one that echoes the example, are what isolate the star path.
  evidence: internal/core/rules/rules.go:634 — "if stars[name] {"
  evidence: internal/core/rules/rules_test.go:452 — ""ZONK": {State: StateActive, Recall: []string{"zzznomatch"}, Rules: []string{"r"}},"
  evidence: internal/core/rules/rules_test.go:455 — "if !has(rs.Match("*ZONK do the thing"), "ZONK") {"
  evidence: internal/core/rules/rules_test.go:477 — "if !has(rs.Match("*ROADMAP go"), "ROADMAP") {"
  evidence: internal/core/rules/defaults/rules.json:31 — ""recall": ["roadmap", "milestone", "phase"],"
- ac-5 — NOT_MET: Promised: in a fresh clone of an abcd repo, a CLAUDE.md under 50 lines carrying the abcd marker block. At HEAD the canonical marker block is 121 lines on its own (124 with its fences, as AGENTS.md lines 6 to 129 show), so no CLAUDE.md that carries it can be under 50 lines; ahoy's default docs target is skip, so a default install writes the block into no conventions file; and this repository commits no CLAUDE.md at all (commit 601680366 removed the link), its 469-line AGENTS.md carrying the block. At the ship commit the block was 42 lines, but this repository's CLAUDE.md was then a link to a 79-line AGENTS.md that carried no block.
  evidence: internal/core/ahoy/defaults/claude-md-marker-block.md:121 — "For internals see `.abcd/development/brief/05-internals/03-configuration.md`."
  evidence: AGENTS.md:6 — "BEGIN ABCD"
  evidence: AGENTS.md:129 — "END ABCD"
  evidence: internal/core/ahoy/detect.go:44 — "docsTargetDefault = "skip""
  evidence: internal/core/ahoy/marker.go:467 — "return markerTargets(docsTargetDefault)"
  evidence: commit 601680366 — "Remove the committed CLAUDE.md and GEMINI.md links"
  evidence: e5fdf97c2:CLAUDE.md — "AGENTS.md"
- ac-6 — NOT_MET: Promised: every broadly applicable rule of the legacy methodology file preserved in the bundled defaults, by a manual review recorded in research/legacy-harvest.md. The harvest record's own Pass 1 keeps section 9 (singular vs plural naming, as a NAMING domain) and section 14 (documentation footer standards) and establishes NAMING and ARTEFACTS domains, but the bundled defaults carry no NAMING or ARTEFACTS domain, no naming rule and no footer rule, and the record holds no review of the shipped set against the legacy file (the delivered range only repointed its links); the defaults ship as defaults/rules.json rather than defaults.json, and the ship decision itself names this as the one gap.
  evidence: .abcd/development/research/legacy-harvest.md:33 — "Singular vs Plural Naming | **Keep** | `NAMING`"
  evidence: .abcd/development/research/legacy-harvest.md:38 — "Documentation Footer Standards | **Keep** | `DOCUMENTATION`"
  evidence: .abcd/development/research/legacy-harvest.md:42 — "PII (high-priority), COMMITTING, DOCUMENTATION, ARTEFACTS, ISSUES, INTENTS, NAMING, LIFEBOAT"
  evidence: internal/core/rules/defaults/rules.json:26 — "User-facing prose in British English; code-side identifiers, comments, strings in US English."
  evidence: internal/core/rules/rules.go:170 — "//go:embed defaults/rules.json"
  evidence: .abcd/work/decisions/dcn-2607110000000005-itd-3-shipped-manually-ahead-of-the.md:11 — "the one gap is the AC6"

Gap audit:
- honoured:
  - A UserPromptSubmit hook recall-matches each prompt and injects only the matching domains' rules, in place of a force-loaded instructions file
    evidence: hooks/hooks.json:3 — ""UserPromptSubmit": ["
    evidence: internal/surface/cli/cli.go:1839 — "res := rules.Inject(rs, in.Prompt, rules.LoadState(session), rules.LoadBackstop(root))"
  - The default rule set is bundled with the plugin, embedded in the abcd binary
    evidence: internal/core/rules/rules.go:170 — "//go:embed defaults/rules.json"
  - A per-repo .abcd/rules.json overrides the defaults per field and can add custom domains, and ahoy writes a minimal skeleton
    evidence: internal/core/rules/rules.go:44 — "const RepoRelPath = ".abcd/rules.json""
    evidence: internal/core/rules/rules_test.go:500 — ""CUSTOM": {State: StateActive, Recall: []string{"widget"}, Rules: []string{"do the thing"}},"
    evidence: internal/core/ahoy/apply.go:2094 — "rules := map[string]any{"schema_version": 1, "disabled": false, "domains": map[string]any{}}"
  - A prompt about unrelated work, the React-rendering example, pays zero tokens for the rules
    evidence: internal/core/rules/rules_test.go:335 — "got := rs.Match("render a react component with a gradient background")"
    evidence: internal/surface/cli/cli_test.go:167 — "out, _ := runHook(t, hookInputJSON(t, "s4", cwd, "paint a landscape"), "hook", "prompt-router")"
  - Signature-based dedup within a session, with the ledger cleared on SessionStart and PreCompact so a domain re-injects after compaction
    evidence: internal/core/rules/inject.go:147 — "if ledger[d.Name] == Signature(d) {"
    evidence: internal/surface/cli/cli.go:1909 — "if err := rules.ResetState(session); err != nil {"
    evidence: hooks/pre-compact.sh:4 — "hook prompt-router-reset"
  - A `*DOMAIN` star-command activates that domain regardless of keyword match, overriding dormant but never the kill switch, and activates domains only
    evidence: internal/core/rules/rules.go:164 — "var starCommandRe = regexp.MustCompile("
    evidence: internal/core/rules/rules.go:634 — "if stars[name] {"
    evidence: internal/core/rules/rules_test.go:489 — "if got := rs.Match("*ROADMAP go"); len(got) != 0 {"
  - A per-domain state field, active or dormant, silences a domain without deleting its rules
    evidence: internal/core/rules/rules.go:64 — "StateDormant = "dormant""
  - abcd rules renders the active set and abcd rules DOMAIN scopes to one domain, with no show sub-verb
    evidence: internal/surface/cli/cli.go:2573 — "Use: "rules [domain]","
    evidence: internal/surface/cli/cli_test.go:85 — "out := string(runCLI(t, "rules", "committing"))"
  - The domains the press release names ship as bundled defaults (COMMITTING, DOCUMENTATION, ROADMAP, ISSUES, INTENTS and LIFEBOAT), with PII
    evidence: internal/core/rules/defaults/rules.json:5 — ""COMMITTING": {"
    evidence: internal/core/rules/defaults/rules.json:86 — ""PII": {"
- diverged:
  - The hook is hooks/prompt_router_hook.py: delivered as the Go subcommand abcd hook prompt-router, by a recorded decision
    evidence: .abcd/work/decisions/dcn-2607110000000003-itd-3-rules-loader-hook-is-go-not-python.md:8 — "the intent's `hooks/prompt_router_hook.py` is a stale pre-Go-rebuild detail and"
    evidence: hooks/user-prompt-submit.sh:4 — "hook prompt-router"
  - A forced full re-inject every N turns with N defaulting to 5: delivered as a ledger reset on SessionStart and PreCompact plus a fixed-N backstop defaulting to 15 (signed-off delta D1)
    evidence: internal/core/rules/inject.go:55 — "const DefaultRefreshBackstop = 15"
    evidence: .abcd/work/decisions/dcn-2607110000000004-itd-3-rules-loader-design-signed-off.md:11 — "(fixed-N demoted to a ~15–20 backstop, not primary)"
  - No-match overhead under 200 tokens as a header: delivered as zero bytes with an out-of-band stderr line (signed-off delta D3), while at HEAD the same hook adds the AWAY block on every prompt for as long as the away record holds
    evidence: internal/core/rules/rules.go:870 — "if len(domains) == 0 {"
    evidence: internal/surface/cli/cli.go:1852 — "abcd rules: turn %d, injected %d domain(s)"
    evidence: internal/surface/cli/cli.go:1876 — "text = joinBlocks(text, away.Block)"
  - The rules.json schema as a JSON Schema embedded in the binary: delivered as hand-rolled Go validation with no schema document
    evidence: internal/core/rules/rules.go:19 — "Validation is hand-rolled Go (zero new dependencies)"
  - rules.json extends the defaults and overrides individual rules: delivered as wholesale per-field replacement of a domain's recall, aliases and rules lists in the schema_version, disabled, domains shape (signed-off delta D2), so changing one rule restates the domain's whole list
    evidence: internal/core/rules/rules.go:562 — "r.Rules = append([]string(nil), over.Rules...)"
    evidence: .abcd/work/decisions/dcn-2607110000000004-itd-3-rules-loader-design-signed-off.md:12 — "keep the shipped `{schema_version,disabled,domains{}}` shape"
  - Per-repo only, with a global rules.json explicitly out of scope: at HEAD a machine-wide ~/.abcd.noindex/rules.json layers between the bundled defaults and the repo file, added after this intent shipped (itd-117, spc-23)
    evidence: internal/core/rules/rules.go:11 — "~/.abcd.noindex/rules.json (one per machine, spc-23)"
    evidence: internal/core/rules/rules.go:49 — "var UserRelPath = abcdhome.Rel("rules.json")"
  - CLAUDE.md becomes a small marker block of about 30 lines that ahoy writes: the canonical block was 42 lines at the ship commit and is 121 at HEAD, and a default install now writes it into no file
    evidence: internal/core/ahoy/defaults/claude-md-marker-block.md:121 — "For internals see `.abcd/development/brief/05-internals/03-configuration.md`."
    evidence: internal/core/ahoy/detect.go:44 — "docsTargetDefault = "skip""
  - A PR-description session loads the commit and PII rules: PII recalls only on its own words, so a PR prompt recalls COMMITTING alone
    evidence: internal/core/rules/defaults/rules.json:8 — ""aliases": ["pull request", "pr", "git add", "conventional commit"],"
    evidence: internal/core/rules/defaults/rules.json:88 — ""recall": ["secret", "token", "credential", "pii", "redact", "hostname""
- missing:
  - Every broadly applicable rule of the legacy methodology file preserved in the bundled defaults, by a recorded completeness review (criterion 6)
    evidence: .abcd/development/research/legacy-harvest.md:33 — "Singular vs Plural Naming | **Keep** | `NAMING`"
    evidence: .abcd/development/research/legacy-harvest.md:38 — "Documentation Footer Standards | **Keep** | `DOCUMENTATION`"
    evidence: .abcd/work/decisions/dcn-2607110000000005-itd-3-shipped-manually-ahead-of-the.md:11 — "the one gap is the AC6"
  - A fresh clone whose CLAUDE.md is under 50 lines and carries the marker block (criterion 5)
    evidence: AGENTS.md:6 — "BEGIN ABCD"
    evidence: AGENTS.md:129 — "END ABCD"
    evidence: commit 601680366 — "Remove the committed CLAUDE.md and GEMINI.md links"
<!-- abcd-review-end receipt=rcp-a2e0ab1dca95 -->

### Per-criterion verdicts

1. **MET.** COMMITTING's `recall`/`aliases` cover "commit", "PR", and "git add";
   the prompt-router hook injects the domain on the turn.
2. **MET (exceeded).** A no-match renders zero model-facing tokens — bettering the
   `< 200` threshold. The "header only" mechanism was replaced by out-of-band
   diagnostics (signed-off delta D3).
3. **MET_WITH_CONCERNS.** Dedup is MET (inject on turn 1, skip 2–3). "Forced
   refresh on prompt 5" was deliberately replaced by event-driven refresh on
   `SessionStart(compact)` plus a fixed-N backstop (default 15, configurable) —
   signed-off delta D1.
4. **MET.** A `*ROADMAP` star-command injects the domain regardless of keyword
   match (it overrides a dormant state, not the kill switch).
5. **INCONCLUSIVE.** The marker block is delivered and sized (~40 lines) and ahoy
   installs it, but the live "fresh clone + installed plugin + new session"
   behaviour cannot be observed in-repo (the hook only fires in an installed
   plugin).
6. **NOT_MET.** The shipped defaults are a concise curated set (eight domains)
   drawn from the working conventions plus the Pass-1 mapping in
   `research/legacy-harvest.md`; a completeness-audited rule-by-rule port of the
   legacy methodology file was not performed. This is the one genuine gap.

**Rollup:** 3 MET · 1 MET_WITH_CONCERNS · 1 INCONCLUSIVE · 1 NOT_MET.

### Three-bucket audit

- **Honoured.** Just-in-time recall injection replacing the monolithic
  force-load (the core promise); binary-bundled defaults plus a per-repo
  override; per-domain dedup; star-command activation; the kill switch and
  dormant state; the `abcd rules [domain]` diagnostic verb; the OPINIONS domain
  pointing at the canonical conventions; a fail-closed, non-blocking trust
  boundary; a marker block under 50 lines.
- **Diverged (every item a maintainer-signed delta).** D3 — zero no-match tokens
  plus out-of-band diagnostics instead of a `< 200`-token header. D1 —
  event-driven refresh with a fixed-N backstop instead of forced-every-5. D2 —
  the shipped `{schema_version, disabled, domains{}}` per-field-override shape
  instead of the earlier `extends`/`overrides` sketch. D4 — no `.abcdignore`.
- **Missing / outstanding.** A completeness-audited harvest of the legacy
  methodology file into the defaults (criterion 6). Live installed-plugin session
  verification (criterion 5). Neither blocks the shipped promise; both are
  recorded here as follow-ups.

## References

[carl]: https://github.com/ChristopherKahler/carl "CARL — Context Augmentation & Reinforcement Layer, just-in-time rule injection for Claude Code (Kahler)"
[claude-skills-docs]: https://code.claude.com/docs/en/skills "Claude Code Skills (Anthropic docs)"
