---
schema_version: 1
id: "iss-2608280807166510"
slug: "validrelpath-is-slash-only-so-a"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "adversarial review of the path-canonicalisation batch (2026-08-28)"
found_at: "internal/fsutil/paths.go"
resolution: "ValidRelPath rejects a backslash segment, closing the Windows backslash-traversal escape shared by every caller"
impact: fix
resolved_by:
  commit: "97c6d6d7"
---

fsutil.ValidRelPath is slash-only, so a backslash traversal escapes on a Windows target: path.Clean leaves backslashes untouched, so '..\\..\\x' passes ValidRelPath and, after filepath.Join/FromSlash on GOOS=windows, can walk out of the intended root. The gap is shared by every ValidRelPath caller (installsurface dirTree, positioning surfaces, lifeboat manifests) and is pre-existing, not introduced by the path-canonicalisation batch. Decide whether Windows is a target that ingests attacker-controlled relative config/manifest paths; if so, ValidRelPath should reject a segment containing filepath.Separator (or any backslash) as well as forward-slash traversal.