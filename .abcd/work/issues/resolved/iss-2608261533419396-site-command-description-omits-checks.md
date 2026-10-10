---
schema_version: 1
id: "iss-2608261533419396"
slug: "site-command-description-omits-checks"
severity: "nitpick"
category: "documentation"
source: "agent-observation"
found_during: "bughunt-a round 9"
found_at: "commands/site.md"
resolution: "The description discloses check's render-if-absent write alongside bare and build"
impact: fix
resolved_by:
  commit: "0458d586"
---

commands/site.md's frontmatter description enumerates write posture for the bare and build forms only, while check — added later with the argument-hint updated in the same diff — renders the whole site into an output directory that has no index.html. The body documents this correctly; the description line is a stale roster, and it drops the counter-intuitive form. House style names every form's posture, surprising ones explicitly (identity, version). Acceptance: the description discloses check's render-if-absent write.