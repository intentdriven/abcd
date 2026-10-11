---
schema_version: 1
id: "iss-2609231526392449"
slug: "guard-hook-reads-abcd-guard-json-from-a"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
resolution: "Resolved on its merits; the wording was the defect, not the read. AGENTS.md 'Foreign-uid roots' states that the refusal bounds the walk, not the working directory, so a session started AT a refused root reads that root's .abcd/ configuration by design, and the posture change that would refuse that read too stays deferred (DECISIONS.md 2026-09-25, the unmerged feat/ruled-security-forks). The note that claimed the root's guard.json was NOT read was corrected at base by ec5ca74f8 (v0.11.1, iss-2609251522588539, iss-2609261753290536): where the working directory carries a real .abcd/ the note says it IS read and governs the session. The guard's per-call workdir goes through the same rules.Resolve (rulesRoot), so the note printed for a workdir at the refused root is computed for that workdir and says the same; a workdir's registry can only add hazards (guard.Strictest), and its text reaches the agent through termsafe.Sanitize. Pinned by TestResolveRootRefusalSaysWhatItStillReads."
impact: fix
shipped_in: v0.11.1
resolved_by:
  commit: "ec5ca74f8"
---

guard hook reads .abcd/guard.json from a REFUSED foreign-uid root when the workdir (or session cwd) is that root, while the refusal note says it was not read (internal/surface/cli/guard.go:430, internal/core/rules/root.go:255); it can only add hazards (Strictest), but the foreign file's why/successor text reaches the agent as the block message. Fixed on feat/ruled-security-forks (da3efea6/ebc6290d), not on main.

## Grounds

- pursued: the refusal note never says a configuration went unread that the loaders read; a session or workdir at a refused root with its own .abcd/ whose note says NOT read would show it wrong
