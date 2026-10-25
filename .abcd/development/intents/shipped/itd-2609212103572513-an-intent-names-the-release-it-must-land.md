---
id: itd-2609212103572513
slug: an-intent-names-the-release-it-must-land
spec_id: spc-2609212138243443
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609211913453478, itd-2609212103568351]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-73]
related_adrs: [adr-2609212115255771]
---

# An intent names the release it must land by, and the cut says whether it did

## Press Release

> **A planned intent may carry `target_release`; the cut reports every targeted intent still unshipped and moves its target forward.**
>
> "Milestones were the only thing that said 'this must be in the next one', and they went with the phases," said a product thinker at a cut. "Now the intent says it, the dry run lists what has not made it, and after the cut the target rolls forward on its own. It never refuses my release; it never lets me forget either."

## Why This Matters

The research pass of 2026-09-21 found the strongest challenge to retiring milestones: a derived release says what shipped, and nothing says what must land before the next cut. Kubernetes answers it with a milestone field on the record rather than a unit; this record is that field, ruled report-and-move-forward, never refuse (adr-2609212115255771, decision 3).

## Mechanism

We expect a target the cut reports and carries forward to answer "what must land" without a planned unit, because the field lives on the record that owns the work and the cut is the moment anyone reads it; shown wrong if targets are never set, or set and carried past two cuts without a word.

## Scope Conditions

None stated.

## What's In Scope

- **The field**: `target_release: vX.Y.Z` (or `next`) on a planned intent, validated as a version, written by `intent plan --target` or `intent target <itd-N> <version>`; the record lint refuses it on a shipped or superseded intent.
- **The report**: `launch --dry-run` and the cut list every targeted intent not yet shipped, in text, in the receipt and in `--json`; the cut proceeds.
- **The move**: at the cut, each unshipped target the cut passes becomes `next` (whatever the following release is numbered) in the change that rolls the changelog, and the changelog names the move.
- **The board**: the status block marks a targeted intent with its target in Next and Later.

## What's Out of Scope

- A refusing gate.
- A target on an issue.
- Any date.

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609212115255771 records the vocabulary rulings it rests on):

1. Report, never refuse (adr-2609212115255771, decision 3).
2. The target moves forward at the cut, so the field never goes stale.
3. The field is optional and lives on the intent alone.
4. A target the cut passes becomes `next`, whatever the following release is numbered, never a version number: the product thinker's ruling BS1 of 2026-09-29 (the dated entry of that day, [dcn-2609290000000007](../../../work/decisions/dcn-2609290000000007-the-product-thinker-answered-twenty-six.md)). The version a cut derives is the one it cuts, so a number written at the cut would name a release already out, which decision 2 exists to prevent. A target the cut passes is `next` (it named this release) or a tag at or below the one cut; a tag above it is still ahead and stays. Criterion 3 is worded to the ruling.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** a planned intent, **when** `intent target <itd-N> v0.11.0` runs, **then** the record carries `target_release: v0.11.0`, and the same on a shipped or superseded intent is refused by the verb and by the lint.
- **Given** a targeted intent still planned, **when** `launch --dry-run` or the cut runs, **then** it is listed as targeted and unshipped in text, the receipt and `--json`, and the cut proceeds.
- **Given** the cut is written, **when** the changelog is rolled, **then** each unshipped target the cut passes becomes `next` (whatever the following release is numbered) in the same change, and the changelog names the move.
- **Given** the status block, **when** a targeted intent is listed, **then** its row shows the target.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-47e25ab4498e -->
Fidelity review — receipt rcp-47e25ab4498e (verifier abcd:intent-auditor claude-opus-5-5).

Provenance: abcd:intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:dd755933747d43eeeaaf8c3c09e2695c35890ca45eb5760ab6b2364798274b86
Input attestations: diff:1ded82939be78d6f5c5f7fba5f457c0e4c4e2684..11c17fd800cb7bfabf885aa7cc97d5b32a44e11f@sha256:5bbdb6a69d548dc7128751ed8be628d776f4f0b7076adffeef52811ba0988d25; merge:a87566e21ed5eac6f7f56ae68bbbf85b206a7086@-; diff:2b9d52fbbfe4a65096b214a6140667fc0657fb7f..279f5241f1425638a54a9a36a94b9f0192441a06@sha256:30b8475ea4e35150969a1fefd05bf2240ab545ed25b9e9db1f0c2beef66b9c3e; merge:0f6c018366b0ea07e6ef3eff8a25c011263a6f2d@-; repository:2e03c61e454bf09ecda8cee0ae2f377157b522f7@-; intent:.abcd/development/intents/shipped/itd-2609212103572513-an-intent-names-the-release-it-must-land.md@sha256:348eb9b428a71df0f48e0aee2d408d4a2126b73fc99b35a6101f6af6fc49f252; spec:.abcd/development/specs/closed/spc-2609212138243443-an-intent-names-the-release-it-must-land.md@sha256:1532388954e69b1099a0050dddcc4cd7f47897ea09c06efb7d1fdb9f0640611d; request:.abcd/.work.local/reviews/rcp-47e25ab4498e.request.md@sha256:3056b1058688241b8483e16032e025b4b1789e7dae54c1394582a6b919b02ec4;

Acceptance rollup: MET 3 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: intent.Target writes `target_release: v0.11.0` onto a planned record as one added frontmatter line (pinned at target_test.go:50), and refuseTargetBucket refuses every other shelf with nothing written; the shipped and superseded cases are among those tested. record_schema's target leg, which the repository's blocker-severity record lint runs, refuses the key on a shipped or superseded intent, and its own test pins that. The verb is wired as `abcd intent target <itd-N> <vX.Y.Z|next>` and documented on commands/intent.md.
  evidence: internal/core/intent/target.go:84 — "updated, err := setFrontmatterFields(content, map[string]string{launch.TargetReleaseKey: version})"
  evidence: internal/core/intent/target.go:110 — "return fmt.Errorf("intent: %s is in %s; a target names the release unshipped work must land by, so it is written only onto a planned intent (nothing written)","
  evidence: internal/core/intent/target_test.go:115 — "{"itd-3", "v0.11.0", "shipped"},"
  evidence: internal/core/lint/schema_target.go:40 — "case "shipped", "superseded":"
  evidence: internal/core/lint/schema.go:757 — "out = append(out, checkIntentTarget(r, cfg.Severity)...)"
  evidence: internal/core/intent/target_test.go:140 — "func TestTheRecordLintRefusesATargetOnAShippedOrSupersededIntent(t *testing.T) {"
  evidence: internal/surface/cli/cli.go:3109 — "Use: "target < itd-N> < vX.Y.Z|next>","
- ac-2 — MET: The preview reads the planned targets (intent.Targets) into DryRunReport.targets, which is its --json, into the 'Targeted, not shipped' section of the pre-flight report it writes, and onto a `targeted: <id> targets <v>, not shipped` line. The cut reads them onto release.Cut.targets before any refusal returns, prints the same line in both the emit and the ingest output, and the ingest's pre-flight report carries them too. Nothing adds a refusal: a store that cannot be read is reported as targets_error. The CLI test runs the preview, the emit step and the written cut to exit 0 with the line present. 'The receipt' is read here as the pre-flight report each run writes.
  evidence: internal/surface/cli/cli.go:490 — "Targets: launchTargets(cwd, cmd.ErrOrStderr()),"
  evidence: internal/core/launch/dryrun.go:97 — "Targets []TargetedIntent `json:"targets,omitempty"`"
  evidence: internal/core/launch/preflight_report.go:173 — "section("Targeted, not shipped", lines)"
  evidence: internal/core/release/emit.go:237 — "if targets, err := intent.Targets(root); err != nil {"
  evidence: internal/surface/cli/ship.go:749 — "fmt.Fprintf(w, " targeted: %s\n", line)"
  evidence: internal/surface/cli/ship.go:579 — "report.Targets = launchTargets(cwd, cmd.ErrOrStderr())"
  evidence: internal/surface/cli/intent_target_cli_test.go:142 — "t.Fatalf("the cut must proceed with a target unshipped: exit = %d\n%s", code, shipped)"
- ac-3 — MET: launch.MissedTargets picks the targets a cut passes: `next`, and any tag at or below the derived tag. release.Ingest is the cut's only production write. Under the intent store's lock it plans each such record's rewrite to `target_release: next` and runs it first, in the same write plan as the release page and the CHANGELOG. The rewrite is undone with them, and a target changed since the cut read it stops the cut. The dated section carries one note naming every move. Tests pin the rewrite, the write order, the note and the undo. The v0.14.2, v0.15.0 and v0.15.1 cut commits each carry the note for the one live target, in the commit that rolled the changelog.
  evidence: internal/core/launch/target.go:85 — "func MissedTargets(targets []TargetedIntent, nextTag string) []TargetMove {"
  evidence: internal/core/release/ingest.go:394 — "section := renderSection(heading, entries, changelog.TargetMoveNote(moves))"
  evidence: internal/core/release/ingest.go:455 — "rewrites, err := intent.PlanTargetMoves(root, moves)"
  evidence: internal/core/intent/target.go:220 — "updated, err := setFrontmatterFields(content, map[string]string{launch.TargetReleaseKey: launch.TargetNext})"
  evidence: internal/core/release/write.go:244 — "for _, rec := range plan.records {"
  evidence: internal/core/release/targetmove_test.go:80 — "if !strings.Contains(log, "## [0.4.1] - 2026-07-21\n\n"+sectionNotice+"\n\n"+note+"\n\n### Added\n") {"
  evidence: CHANGELOG.md:23 — "Targeted and not shipped in this release, so each targets the next release (`next`): itd-2610112239022766 (targeted `next`)."
- ac-4 — MET_WITH_CONCERNS: statusblock.Read gives every planned intent's row its target in Now, Next and Later, and the JSON carries it as target_release. The facilitator's text board adds `target <value>` to a Now or Next row's brackets. The site's Status page adds it after what places a row, Later rows included. Concerns: the in-scope line promised the mark in Next and Later, but the text board draws Later as a count alone (ruling BV1), so a Later intent's target appears only in --json and on the site. Also, since spc-2610031844142274 (2026-10-05), bare `abcd` and the plugin command open on the product thinker's view. That view lists intents by title and shows no target by design, so the text target appears only under `--view facilitator`.
  evidence: internal/core/statusblock/statusblock.go:195 — "r.Target = it.TargetRelease"
  evidence: internal/core/statusblock/statusblock_test.go:614 — "if !strings.Contains(string(raw), `"id":"itd-8","title":"The unlinked one","bucket":"planned","target_release":"next"`) {"
  evidence: internal/core/board/board.go:387 — "parts = append(parts, "target "+termsafe.Sanitize(r.Target))"
  evidence: internal/surface/cli/board_status_test.go:249 — "" itd-5 The head [next up; target next]\n","
  evidence: internal/core/site/status.go:75 — "target := escapeText(e.c.ui.Status.Target) + ` ` + escapeText(r.Target)"
  evidence: internal/core/board/board.go:308 — "es = append(es, entry{2, fmt.Sprintf("Later: %d %s", len(b.Later), noun)})"
  evidence: internal/surface/cli/board_view.go:28 — "return board.Product, nil"

Gap audit:
- honoured:
  - A planned intent may carry `target_release`. It is written by `intent target` or `intent plan --target`, validated as `next` or a bare vX.Y.Z tag through one shared predicate, and refused on a shipped or superseded intent by both the verb and record_schema. Every move out of planned/ drops the line.
    evidence: internal/core/launch/target.go:33 — "func ValidTargetRelease(value string) error {"
    evidence: internal/core/intent/target.go:63 — "if err := refuseTargetBucket(it); err != nil {"
    evidence: internal/core/lint/schema_target.go:40 — "case "shipped", "superseded":"
    evidence: internal/core/intent/lifecycle.go:1446 — "return dropTarget(updated)"
  - The dry run lists what has not made it: every targeted planned intent appears in its text, in its --json `targets` and in its pre-flight report, and none adds a refusal.
    evidence: internal/surface/cli/cli.go:490 — "Targets: launchTargets(cwd, cmd.ErrOrStderr()),"
    evidence: internal/core/launch/preflight_report.go:173 — "section("Targeted, not shipped", lines)"
    evidence: internal/core/launch/target_test.go:58 — "t.Fatalf("a target must never add a refusal:\n without: %v\n with: %v", baseline.WouldRefuseOn, rep.WouldRefuseOn)"
  - The cut reports every targeted intent still unshipped and never refuses on one. The list is read before any refusal returns, printed as `targeted:` lines, and carried as `targets` in the JSON and in the ingest's pre-flight report.
    evidence: internal/core/release/emit.go:237 — "if targets, err := intent.Targets(root); err != nil {"
    evidence: internal/surface/cli/ship.go:749 — "fmt.Fprintf(w, " targeted: %s\n", line)"
    evidence: internal/surface/cli/ship.go:579 — "report.Targets = launchTargets(cwd, cmd.ErrOrStderr())"
  - After the cut the target rolls forward on its own and is never forgotten. The ingest moves each passed target to `next` in the write that rolls the changelog, prints a `moved:` line, and the dated section names the move. Every cut made while a target was set (v0.14.2, v0.15.0 and v0.15.1) named it.
    evidence: internal/core/release/ingest.go:394 — "section := renderSection(heading, entries, changelog.TargetMoveNote(moves))"
    evidence: internal/core/release/ingest.go:455 — "rewrites, err := intent.PlanTargetMoves(root, moves)"
    evidence: internal/surface/cli/ship.go:813 — "fmt.Fprintf(w, " moved: %s targets next (targeted %s)\n", termsafe.Sanitize(m.ID), termsafe.Sanitize(m.From))"
    evidence: CHANGELOG.md:23 — "Targeted and not shipped in this release, so each targets the next release (`next`): itd-2610112239022766 (targeted `next`)."
- diverged:
  - The board was promised to mark a targeted intent with its target in Next and Later. The row carries the target in Now, Next and Later in --json and on the site's Status page. The text board, however, draws Later as a count (ruling BV1), so a Later intent's target is not on it. Since spc-2610031844142274, bare `abcd` and the plugin command also open on the product thinker's view, which shows no target by design, so the text mark appears only under `--view facilitator`.
    evidence: internal/core/board/board.go:308 — "es = append(es, entry{2, fmt.Sprintf("Later: %d %s", len(b.Later), noun)})"
    evidence: internal/surface/cli/board_view.go:28 — "return board.Product, nil"
    evidence: internal/core/board/board_test.go:103 — "if handle.MatchString(l) || command.MatchString(l) || strings.Contains(l, "waiting on") || strings.Contains(l, "target") {"
    evidence: .abcd/development/brief/04-surfaces/08-abcd.md:47 — "id, lane, target or command in it."
  - The move was filed as 'rewritten to the next version' but is delivered as the symbolic `next`, never a number. This follows ruling BS1, the intent's decision 4; criterion 3 and the in-scope line were reworded in the ship commit 279f5241f. A target already `next` is not rewritten, so a missed target reads `next` cut after cut. The record keeps no trace of the release it missed; only the changelog note does. itd-2610112239022766 has been named in three consecutive cuts (v0.14.2, v0.15.0, v0.15.1) while its record still reads `target_release: next`. Reflect's seed matches target_release to the release tag, so it no longer sees an intent a cut passed.
    evidence: internal/core/intent/target.go:217 — "if current == launch.TargetNext {"
    evidence: CHANGELOG.md:98 — "Targeted and not shipped in this release, so each targets the next release (`next`): itd-2610112239022766 (targeted `next`)."
    evidence: .abcd/development/intents/planned/itd-2610112239022766-when-someone-reports-a-security-hole-in.md:14 — "target_release: next"
    evidence: internal/core/reflect/seed.go:173 — "if it.Bucket == intent.BucketPlanned && it.TargetRelease == tag {"
- missing: (none)
<!-- abcd-review-end receipt=rcp-47e25ab4498e -->

Changed on 2026-09-30 by the product thinker's ruling TG1 of that day, recorded as adr-2609301720596683: a managed repository's `site-src/ui.json` written before this intent's `status.target` label existed keeps building, because `abcd site setup` and `abcd site build` add each declared label the file lacks with abcd's default words, name it on stderr, and change nothing else in the file. The impact stays `additive`.

## Grounds

- pursued: milestones are retired today and this is the only place must-land-by survives; we expect the cut's report to be read and the moved target to be acted on; shown wrong if targets are never set or carried past two cuts unremarked
