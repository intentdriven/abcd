---
schema_version: 1
id: "iss-2610010618373016"
slug: "managed-agents-md-block-names-paths-a"
severity: "minor"
category: "bug"
source: "managed-repo"
found_during: "managed-repo report relayed by a peer session, 2026-10-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/defaults/claude-md-marker-block.md"
remedy: "Rewrite the two pointers in claude-md-marker-block.md so the managed block names only paths that exist in a managed repository, or names them as abcd-repository locations (a URL to abcd's published docs); prove it with a test that every repo-relative path in the rendered block resolves in a freshly adopted fixture repository."
---

The conventions-file block ahoy writes into a managed repository's AGENTS.md / CLAUDE.md (template internal/core/ahoy/defaults/claude-md-marker-block.md) names two paths that exist only in abcd's own source tree: .abcd/development/principles/ (the OPINIONS paragraph, 'the canonical conventions under .abcd/development/principles/') and .abcd/development/brief/05-internals/03-configuration.md (the closing 'For internals see …' line). Both sit inside the abcd-managed fence, so the adopter cannot hand-fix them: ahoy rewrites the block on drift. Met in practice in a managed repository whose CI pins v0.11.1, relayed by a peer session; that repository holds its own record deferred until abcd ships the fix. The principles half shares its ruling with iss-2608210934566220 (the injected OPINIONS rules' dangling pointers); the 03-configuration.md pointer is independent of that ruling and fixable on its own. Expected: the managed block names only paths that exist in the managed repository, or names them as abcd-repository locations.
