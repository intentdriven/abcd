---
id: itd-102
slug: your-repo-says-the-same-thing-about
spec_id: spc-19
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
---

# **Your repo says the same thing about itself everywhere — because abcd asked you once and holds every surface to it.** A project's positioning fragments silently: the README strapline, the package or plugin manifest description, and the agent conventions file are each edited at different moments, and soon three surfaces tell three stories about what the project is. When abcd prepares a repo it interviews the user for the project's identity — title, tagline, and a short elevator pitch — and records the answers in one canonical home in the development record. Every surface rendering derives from that home, and a deterministic positioning check compares the surfaces against it on every audit: a mismatch is highlighted with the exact drifted line, and the fix is either re-render from the record or a deliberate, recorded identity change — never a silent extra variant, and never a silent rewrite by abcd. "I changed my tagline once, in one place, and abcd chased it everywhere else," said Alice, a solo founder. "When a stray edit crept into the README, the next audit pointed at the exact line instead of letting the drift settle in."

## Press Release

> **Your repo tells one story about itself, everywhere — because abcd asks you once and holds every surface to the answer.** When abcd prepares a repository it interviews the maintainer for the project's identity — its title, its tagline, and a short elevator pitch — and records those answers in a single canonical home in the development record. Every rendered surface, from the README strapline to the plugin manifest description to the agent conventions file, derives from that one home rather than being retyped, and a deterministic positioning check compares each surface against it on every audit. Drift is caught at the exact line: the fix is either a re-render from the record or a deliberate, recorded change to the identity itself — never a silent third variant, and never a rewrite abcd makes on its own.

## Why This Matters

**Your repo says the same thing about itself everywhere — because abcd asked you once and holds every surface to it.** A project's positioning fragments silently: the README strapline, the package or plugin manifest description, and the agent conventions file are each edited at different moments, and soon three surfaces tell three stories about what the project is. When abcd prepares a repo it interviews the user for the project's identity — title, tagline, and a short elevator pitch — and records the answers in one canonical home in the development record. Every surface rendering derives from that home, and a deterministic positioning check compares the surfaces against it on every audit: a mismatch is highlighted with the exact drifted line, and the fix is either re-render from the record or a deliberate, recorded identity change — never a silent extra variant, and never a silent rewrite by abcd. "I changed my tagline once, in one place, and abcd chased it everywhere else," said Alice, a solo founder. "When a stray edit crept into the README, the next audit pointed at the exact line instead of letting the drift settle in."

## Acceptance Criteria

- Given repo onboarding runs (the install or prepare flow), when the identity interview completes, then title, tagline, and pitch land in a parseable markdown identity block (fixed bullet shape) whose location the repo's abcd config records — markdown stays the single source of truth.
- Given a rendered surface (README strapline, manifest description, conventions-file opening) diverges from the identity block, when the positioning check runs, then a warn-tier finding names the exact drifted line; per-repo config may upgrade the finding to blocker.
- Given drift is found, then abcd never rewrites a surface autonomously — re-rendering from the block is always a proposed diff a human adopts.
- Given abcd-cli itself, then the check points at the brief product chapter's existing Identity section unchanged, and iss-143 (the three-variant drift) is the acceptance corpus the check must catch.

## Open Questions

- Interview wording and which surfaces are registered for checking by default beyond the canonical three.
- Whether the pitch is required or optional at onboarding (title and tagline are required).

## Grill Settlements (2026-07-27)

- Identity home is a parseable markdown block, not a structured JSON file — consistent with markdown-as-single-source-of-truth; the config records only where the block lives.
- The check is warn-tier by default (highlight, never gate) and per-repo upgradeable; autonomous rewriting is permanently out.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-52648dea351e -->
Fidelity review — receipt rcp-52648dea351e (verifier abcd:intent-fidelity-reviewer claude-fable-5).

Provenance: abcd:intent-fidelity-reviewer@claude-fable-5 · rubric_hash sha256:d401343707d6fa0f8fabad4c67380b1ec356ffd47f1b72a4e1b56ed320e25a4a · prompt_hash sha256:95792472ae74ca0469f69a51c618946e0d33cb1380032460099ed4b469d67e86
Input attestations: diff:PR#158 feat/itd-102-repo-identity @ ca1459805dd4cc3d48adc39433bbd34b098736dd (base main): internal/core/positioning/, internal/core/audit/rule_positioning.go, internal/surface/cli/identity.go, commands/abcd/identity.md, commands/abcd/prepare-this-repo.md, .abcd/positioning.json, internal/fsutil os.Root containment@-; rubric:.abcd/.work.local/reviews/rcp-52648dea351e.request.md@sha256:d401343707d6fa0f8fabad4c67380b1ec356ffd47f1b72a4e1b56ed320e25a4a;

Acceptance rollup: MET 4 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: The onboarding interview (prepare-this-repo, detect-then-interview) calls `abcd identity init`, which writes the identity into a fixed-bullet markdown block and records its location in the repo's positioning registry — markdown is the single source of truth, the registry only points at it.
  evidence: internal/core/positioning/init.go:159 — "fmt.Fprintf(&b, \"- **Title:** %s\\n\", title)  … - **Tagline:** … - **Pitch:**  (fixed bullet shape)"
  evidence: internal/core/positioning/init.go:141 — "if err := writeConfig(r, DefaultConfig(loc)); err != nil  — records where the block lives in ConfigRelPath"
  evidence: commands/abcd/prepare-this-repo.md:118 — "Identity — detect first, interview only if there is nothing to detect … abcd identity init --title \"…\" --tagline \"…\" [--pitch \"…\"]"
  evidence: internal/core/positioning/block.go:85 — "bulletRe = regexp.MustCompile(`^ {0,3}[-*]\\s+\\*\\*([A-Za-z]+):\\*\\*\\s*(.*)$`) — parses the fixed bullet shape back out"
- ac-2 — MET: checkSurface flags a diverged surface StatusDrifted; the audit rule emits a finding naming the file, the 1-based line, the exact drifted text and the canonical line; it is SeverityWarn by default and severityFor upgrades the whole family to SeverityError when the registry sets severity=blocker — both halves are test-enforced (TestPositioningDriftIsAWarnFindingNamingTheExactLine, TestPositioningSeverityUpgradesToBlocker) and passed.
  evidence: internal/core/audit/rule_positioning.go:87 — "surface %q no longer carries the canonical %s — it says %q; the identity block (%s) says %q  (File+Line set from s.File/s.Line)"
  evidence: internal/core/audit/rule_positioning.go:107 — "func severityFor … if EqualFold(cfg.Severity, SeverityBlocker) { return SeverityError }  default SeverityWarn"
  evidence: cmd: go test -run 'TestPositioningSeverityUpgradesToBlocker' ./internal/core/audit/ (via full pkg run) — "ok github.com/REPPL/abcd-cli/internal/core/audit  — AC2 second half: per-repo config upgrades the whole family to blocker"
- ac-3 — MET: The only re-render path is Propose, which returns unified diffs and performs no writes ('It performs no writes whatsoever'); the CLI `abcd identity render` writes nothing and no flag makes it, and the command doc states it plainly — a deliberate identity change is an edit to the block, never an autonomous rewrite.
  evidence: internal/core/positioning/render.go:48 — "It performs no writes whatsoever."
  evidence: internal/surface/cli/identity.go:51 — "Print the proposed correction for every drifted surface as a unified diff (writes nothing)"
  evidence: commands/abcd/identity.md:43 — "prints a unified diff per drifted surface. It **writes nothing**, and no flag"
- ac-4 — MET: abcd-cli's own .abcd/positioning.json points the block at the brief product chapter's pre-existing 'Identity (canonical)' section in .abcd/development/brief/01-product/README.md, which PR#158 does not touch (its last change was the earlier iss-143 consolidation commit 48a3524, not this positioning work); and the iss-143 three-variant drift corpus is captured verbatim and caught — one-at-a-time and all-three-together tests pass.
  evidence: .abcd/positioning.json:4 — "\"file\": \".abcd/development/brief/01-product/README.md\", \"heading\": \"Identity (canonical)\""
  evidence: cmd: gh pr diff 158 --name-only | grep 01-product — "NOT in PR diff — the brief Identity section is pointed at unchanged (last touched by 48a3524, not PR#158)"
  evidence: internal/core/positioning/check_test.go:140 — "func TestCheckCatchesTheIss143DriftCorpus — the three real drifted variants (readme-strapline, manifest-description, conventions-opening) each caught StatusDrifted"
  evidence: cmd: go test -run 'TestCheckCatchesTheIss143DriftCorpus|TestCheckCatchesAllThreeVariantsTogether' -v ./internal/core/positioning/ — "--- PASS: TestCheckCatchesTheIss143DriftCorpus / --- PASS: TestCheckCatchesAllThreeVariantsTogether (Drifted()==3)"

Gap audit:
- honoured:
  - One canonical markdown identity block is the single home; the config records only where it lives
    evidence: internal/core/positioning/init.go:151 — "func blockSection … The single recorded home for how this project names itself"
    evidence: .abcd/positioning.json:2 — "\"block\": { \"file\": …, \"heading\": \"Identity (canonical)\" }"
  - Every registered surface is compared to the block on every audit; a mismatch names the exact drifted line and the canonical line
    evidence: internal/core/audit/rule_positioning.go:87 — "it says %q; the identity block (%s) says %q"
    evidence: internal/core/positioning/check.go:141 — "res.Status = StatusOK … if len(res.Missing) > 0 { res.Status = StatusDrifted }"
  - The fix is a proposed diff or a deliberate recorded block change — never a silent extra variant, never a silent rewrite
    evidence: internal/surface/cli/identity.go:164 — "%d proposed change(s), nothing written. Adopt what you agree with."
    evidence: internal/core/positioning/init.go:91 — "ErrAlreadyAdopted … change the block there, or edit %s deliberately — re-running onboarding never repoints the canon"
  - iss-143's three-variant tagline drift is the acceptance corpus and is caught
    evidence: internal/core/positioning/check_test.go:55 — "The iss-143 acceptance corpus: the three real drifted variants the repo was carrying"
  - Rule is wired into the production audit path (not dead scaffolding)
    evidence: internal/core/audit/rules.go:12 — "identityPositioning{}  — registered in DefaultRules()"
    evidence: internal/surface/cli/cli.go:97 — "root.AddCommand(newIdentityCommand(&asJSON))"
- diverged:
  - Onboarding is described as an install-or-prepare interview, but abcd currently onboards only via the prepare-this-repo bridge (host asks the questions); there is no separate install-time interview beyond the CLI `identity init` it drives — a narrower-than-worded but functionally complete path
    evidence: internal/core/positioning/init.go:35 — "InitRequest is the onboarding interview's outcome. The host asks the questions (the prepare surface carries the wording); this is what it hands over."
- missing: (none)
