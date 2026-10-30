# Publish Implementation Spec

> **Purpose.** This file is a hand-off document for the next agent session.
> Path A (manual first publish) shipped `v0.0.1-bootstrap` from abcdDev → abcd
> on 2026-05-04. This spec describes how to build the *reproducible* version
> using the existing scripts/scan.py + scripts/audit.py architecture.
>
> **Read first:** `04-surfaces/04-launch.md` (the brief's full launch design),
> `idelphiDev/scripts/sync-to-public.sh` (the working precedent), and
> [`publish-prior-art-sync-script.md`](publish-prior-art-sync-script.md)
> (a divergent port of that script, harvested out of `abcdDev/scripts/` —
> records which mechanics are reusable and which are superseded).

## What this builds

Three files, mirroring the existing scan/audit pattern:

```
scripts/
├── publish.py                          ← first-class user-facing entry (NEW)
└── abcd/
    ├── src/
    │   └── publish.py                  ← implementation module (NEW)
    ├── defaults/
    │   └── publish.json                ← bundled defaults (NEW)
    └── schemas/
        └── publish.schema.json         ← schema (NEW)
```

Mirrors `scan.py` / `audit.py` exactly — top-level wrapper imports the
implementation module from `abcd.src.publish`. No subprocess shell-out for
the orchestration; reuse `audit.py` as a sub-step (it already shells out to
gitleaks + scan.py per `audit.json`).

## What problem it solves

Today (post-Path A), publishing abcdDev → abcd is a manual sequence:

1. clean stale files in public
2. copy README/LICENSE/docs from dev
3. scan with scan.py
4. scan with gitleaks
5. commit + tag + push

The reproducible version automates 1-5 with a config-driven manifest, version
auto-increment, and a single command. Substrate for `/abcd:launch` (per brief
§ 04-launch) when it ships — `launch.py` will extend `publish.py` with full
Pass A/B/C agent integration, mirror modes, marketplace.json bumps, etc.

## Architecture (mirrors scan.py + audit.py)

### scripts/publish.py

Thin wrapper, ~30 lines. Identical pattern to `scan.py` / `audit.py`:

```python
#!/usr/bin/env python3
"""abcd publish — first-class entry point for abcdDev → abcd public release."""
from __future__ import annotations

import sys
from pathlib import Path

_HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(_HERE))

from abcd.src.publish import main  # noqa: E402

if __name__ == "__main__":
    sys.exit(main())
```

### scripts/abcd/src/publish.py

The implementation. Structured as:

1. **Config loading** — same precedence pattern as pii.py:
   - Built-in fallback < plugin defaults (`scripts/abcd/defaults/publish.json`)
     < per-repo override (`.abcd/config/publish.json`) < `--config <path>`
2. **Identity probe** — reuse `abcd.src.pii.probe_identity()` for git config
3. **Preflight stages** (configurable, sequenced):
   - Manifest exists and is valid
   - Manifest-listed files have no uncommitted changes in dev repo
   - Public repo exists at expected sibling path (`../<name-without-Dev>/`)
   - Public repo has clean working tree
   - Target tag does not exist in public repo
   - Run `audit.py` (gitleaks history + worktree + scan.py) on dev — abort
     if any hard_fail finding
4. **Copy stage**:
   - Wipe each manifest path in public (only paths in the manifest, not the
     whole tree — preserves public-side files like `.github/`, `CHANGELOG.md`)
   - Copy each manifest entry from dev → public
   - Run scan.py + gitleaks on public AFTER copy (catch anything that snuck
     through despite preflight-on-dev)
5. **Commit + tag stage**:
   - `git add -A` in public
   - `git commit -m "release: <version>"` (configurable template)
   - `git tag -a <version> -m "Release <version>"`
   - DO NOT push automatically — print push command, leave to user
6. **Report** — write JSON + MD report to `.abcd/logbook/publish/<timestamp>/`
   with manifest, files copied, scan results, version, public commit SHA.

### scripts/abcd/defaults/publish.json

```json
{
  "schema_version": 1,
  "_comment": "abcd-bundled publish defaults. Per-repo overrides go in .abcd/config/publish.json.",

  "public_repo_pattern": "{dev_repo_dir}/../{public_name}",
  "public_name_strategy": "strip_dev_suffix",
  "_comment_naming": "abcdDev → abcd; idelphiDev → idelphi; armatureDev → armature. Lowercase strip_dev_suffix removes trailing 'Dev' (case-sensitive).",

  "manifest": {
    "_comment": "Public payload — files and directories copied to the public sibling on each release. Paths are relative to dev-repo root. Directories are copied recursively. Lines starting with # in include/exclude are ignored at runtime; this is JSON so use _comment fields instead.",
    "include": [
      "README.md",
      "LICENSE",
      "docs"
    ],
    "exclude_patterns": [
      ".git/",
      ".abcd/",
      ".flow/",
      ".specstory/",
      ".work/",
      ".DS_Store",
      "*.bak"
    ]
  },

  "version": {
    "strategy": "auto_increment_patch",
    "_comment_strategy": "auto_increment_patch | auto_increment_minor | auto_increment_major | manual. auto_increment_patch is the default per the brief's launch design (§ 10.4). manual mode requires --version on the CLI.",
    "first_version": "v0.0.1-bootstrap",
    "_comment_first": "Used when public repo has no prior tags. Allows pre-1.0 'bootstrap' or 'pre' suffixes.",
    "tag_format": "v{major}.{minor}.{patch}",
    "_comment_format": "After first_version, subsequent tags follow this format. Suffixes (-pre, -beta, -rc) require manual mode."
  },

  "preflight": {
    "_comment": "Each stage runs in sequence; first hard_fail stops the publish. Stages with severity=warn report and continue.",
    "stages": [
      {
        "name": "manifest_exists",
        "label": "Manifest is present and parseable",
        "severity": "hard_fail"
      },
      {
        "name": "dev_clean_for_manifest",
        "label": "Dev repo manifest paths have no uncommitted changes",
        "severity": "hard_fail"
      },
      {
        "name": "public_exists",
        "label": "Public sibling repo exists at expected path",
        "severity": "hard_fail"
      },
      {
        "name": "public_clean",
        "label": "Public repo has clean working tree",
        "severity": "warn",
        "_comment": "Path A had stale untracked .flow/.specstory in public; auto-cleaning these is reasonable. Configurable per-repo."
      },
      {
        "name": "tag_unique",
        "label": "Target version tag does not yet exist in public",
        "severity": "hard_fail"
      },
      {
        "name": "scan_audit_history",
        "label": "scripts/audit.py reports clean (gitleaks history + worktree + scan.py)",
        "severity": "hard_fail",
        "command": ["python3", "scripts/audit.py", "--quiet"],
        "expect_exit_code": 0
      }
    ]
  },

  "copy": {
    "wipe_strategy": "manifest_only",
    "_comment_wipe": "manifest_only | clean_slate. manifest_only removes only the paths listed in manifest.include from public before copying (preserves public-native files like .github/, CHANGELOG.md). clean_slate removes everything except .git/. Default: manifest_only — safer."
  },

  "post_copy_scan": {
    "_comment": "After copying to public but before committing: run scan.py + gitleaks on the public payload. Catches anything that slipped through preflight (e.g., a file that was clean in dev but became leaky after path rewriting).",
    "enabled": true,
    "scanners": ["scripts/scan.py", "gitleaks"],
    "severity": "hard_fail"
  },

  "commit": {
    "message_template": "release: {version}\n\n{body}",
    "body_template": "Automated publish from {dev_repo_name} via scripts/publish.py.\n\nManifest: {manifest_summary}\nDev SHA: {dev_sha}\nDev tag (if any): {dev_tag}",
    "auto_push": false,
    "_comment_auto_push": "Per brief § 04-launch, push is always manual to give human a final review. publish.py prints the push command and exits. Set true to skip the manual gate (NOT RECOMMENDED)."
  },

  "report": {
    "dir": ".abcd/logbook/publish",
    "format": "json+md",
    "_comment": "JSON has full manifest + scan results + commit metadata. MD is human-skim summary, rendered from JSON via scripts/abcd/src/render.py (existing per brief § 03 plugin shape, when it lands)."
  }
}
```

### scripts/abcd/schemas/publish.schema.json

JSON-Schema mirroring the structure above. Required fields, enum values, etc.
Same shape as `scan.schema.json` and `audit.schema.json`. Validate via
`jsonschema` lib at config-load time, fall back gracefully if unavailable.

## CLI contract

```bash
# Default: auto-increment patch from last tag
python3 scripts/publish.py

# Explicit version (overrides auto-increment)
python3 scripts/publish.py --version v0.1.0-beta.1

# Dry-run: full preflight + scan, no copy/commit
python3 scripts/publish.py --dry-run

# Skip post-copy scan (NOT RECOMMENDED)
python3 scripts/publish.py --no-post-scan

# Auto-push (NOT RECOMMENDED — bypasses human review gate)
python3 scripts/publish.py --auto-push

# Custom config
python3 scripts/publish.py --config path/to/custom-publish.json

# JSON report to specific path
python3 scripts/publish.py --json-out .abcd/logbook/publish/20260504.json
```

Exit codes:
- 0: published successfully (or --dry-run completed clean)
- 1: preflight or scan hard_fail; nothing copied/committed
- 2: config error or unrecoverable failure (e.g., public repo not found)

## Auto-versioning logic

Implements the user's "automatic versioning, starting from v0.0.1-bootstrap"
rule:

1. **First publish** (no tags in public repo): use `first_version`
   from config (default `v0.0.1-bootstrap`).
2. **Subsequent publish, no `--version`**: parse the latest tag in public
   that matches `tag_format`. Apply `version.strategy`:
   - `auto_increment_patch`: `v0.1.4` → `v0.1.5`
   - `auto_increment_minor`: `v0.1.4` → `v0.2.0`
   - `auto_increment_major`: `v0.1.4` → `v1.0.0`
3. **`--version <X>`**: use literal X. Validates against
   `^v\d+\.\d+\.\d+(-[a-zA-Z0-9._-]+)?$`. Refuses if X already exists.
4. **Pre-release suffixes**: if previous tag has a suffix like `-bootstrap`,
   `-pre.1`, `-beta.2`, the auto-increment STRIPS the suffix on next bump
   (since `v0.0.1-bootstrap` → `v0.0.2` is the natural progression). To keep
   a suffix, use `--version` explicitly.

## Migration path from Path A

Path A established the *contract* (the manual sequence + the v0.0.1-bootstrap
precedent). publish.py automates that contract. To migrate:

1. **Build publish.py** per this spec (test against abcdDev → abcd).
2. **Run `--dry-run`** to confirm the same files would be copied (compare
   against Path A's commit `9c90e2f` in <personal account>/abcd).
3. **Run live publish** for v0.0.2 (or whatever the next bump is).
4. **Verify** the resulting commit + tag in <personal account>/abcd matches expectation.
5. **Document** the migration in this file (mark Path A complete + link to
   the publish.py-driven release commit).

## Reuse of existing scripts

publish.py should NOT reimplement what scan.py / audit.py already do.
Specifically:

- **PII / secret scanning** — call `python3 scripts/audit.py --quiet` as a
  subprocess (orchestrator pattern, same as audit.py uses to call gitleaks
  and scan.py). Configurable via the `preflight.stages.scan_audit_history`
  block. If audit.py exits non-zero, publish.py aborts.
- **Identity probe** — `from abcd.src.pii import probe_identity` (importable
  module, no subprocess overhead).
- **Config loading pattern** — `from abcd.src.pii import _merge_config` for
  the JSON precedence merge logic (rename `_merge_config` to `merge_config`
  to make it module-level public, then publish.py imports it).
- **Report writing** — when `scripts/abcd/src/render.py` lands per brief
  § 03 plugin shape, use it for JSON → MD rendering. For now, `json.dumps`
  + a simple template string.

## Edge cases the spec handles

1. **Public repo already has the target tag.** Hard fail at preflight stage
   `tag_unique`. Don't auto-bump beyond — let the user decide whether to
   skip or specify `--version` for a different one.
2. **Manifest references file that doesn't exist in dev.** Warn and skip
   (per idelphiDev's pattern). Configurable: `manifest.missing_strategy =
   "warn" | "fail" | "skip_silent"`.
3. **Public repo has untracked files outside the manifest.** Don't touch
   them (per `copy.wipe_strategy = manifest_only`). They might be
   public-native (e.g., GitHub Actions config). Logged in the report.
4. **Dev repo has uncommitted changes outside the manifest.** Don't fail —
   the publish only cares about manifest paths. (Configurable via
   `preflight.stages.dev_clean_for_manifest.scope = "manifest" | "all"`.)
5. **Filter-repo aftermath in public.** If public was history-rewritten and
   has a re-attached origin, normal push works. publish.py doesn't need to
   handle this — it commits + tags locally; the user pushes.

## Acceptance (in the abcd Given-When-Then style)

- **Given** abcdDev with no `.abcd/config/publish.json` override and the
  bundled defaults, **when** `python3 scripts/publish.py --dry-run` runs,
  **then** the output reports the next version (`v0.0.2` after Path A's
  v0.0.1-bootstrap), the manifest payload, scan results, and exits 0
  without touching any files.
- **Given** abcdDev with a manifest file that's been edited but not
  committed, **when** `python3 scripts/publish.py` runs, **then** preflight
  hard-fails at `dev_clean_for_manifest` and prints the dirty file path.
- **Given** abcdDev clean and tag `v0.0.2` already in <personal account>/abcd, **when**
  `python3 scripts/publish.py` (auto-incrementing to v0.0.2), **then**
  preflight hard-fails at `tag_unique` with a hint to use `--version`.
- **Given** abcdDev with a deliberate PII fixture (e.g., a real email in
  README.md), **when** publish.py runs, **then** preflight hard-fails at
  `scan_audit_history` (audit.py exit 1), no copy happens.
- **Given** a clean publish, **when** publish.py completes successfully,
  **then** the public repo has a new commit with message `release: vX.Y.Z`,
  an annotated tag `vX.Y.Z`, and a JSON report in
  `.abcd/logbook/publish/<timestamp>/publish-report.json` documenting the
  manifest, files copied, scans run, and SHAs.
- **Given** publish.py succeeds, **when** the user reads stdout, **then**
  the final lines say "Push when ready: `git -C ../abcd push origin main
  --tags`" and the script exits 0 — push is always manual.

## Brief deltas to log alongside this work

When publish.py lands, fold these into the brief (`.work/issues.md` is the
staging buffer):

1. **Update `.abcd/development/brief/04-surfaces/04-launch.md`** — add a
   "Substrate" section pointing at scripts/publish.py as the substrate
   for the full launch surface. The launch agent stack (launch-gatekeeper,
   documentation-auditor, mirror-modes, marketplace.json) builds on top.
2. **Update `.abcd/development/brief/05-internals/03-configuration.md § 3`
   plugin shape** — add `scripts/abcd/src/publish.py` to the modules list
   and `scripts/abcd/defaults/publish.json` to the defaults list.
3. **Add a `04-surfaces/07-publish.md`** (NEW surface file) — or fold into
   launch.md. Recommendation: fold, because publish IS launch's substrate;
   they're not separate surfaces.

## Estimated effort for next session

- ~150 lines for `publish.py` (top-level wrapper is 30; module is 120-150)
- ~80 lines for `publish.json` (defaults)
- ~60 lines for `publish.schema.json`
- Tests: dry-run against abcdDev → abcd; live-run for v0.0.2.
- One commit. Pre-commit hooks should pass (scan + gitleaks already gating).

This is a focused, single-session deliverable.
