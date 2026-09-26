---
schema_version: 1
id: "iss-2609251827294854"
slug: "renderpayload-hardcodes-the-dirty-tree-gate-to-dirtyskip-for"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/render.go"
resolution: "PayloadRenderRequest carries a DirtyPolicy whose zero value refuses a dirty tree; the ship's post-write render and launch archive --verify state DirtySkip, launch archive without --verify refuses a dirty tree, and the render-path doc-auditor row reports not_measured (TestRenderPayloadRefusesADirtyTreeByDefault, TestLaunchArchiveWithoutVerifyRefusesADirtyTree, TestRenderPathDocAuditRowSaysItWasNotMeasured)."
impact: fix
resolved_by:
  commit: "6628cd14"
---

RenderPayload hardcodes the dirty-tree gate to DirtySkip for every caller (internal/core/launch/render.go), with a comment asserting the caller ran the gate earlier, which RenderPayload cannot know: the two live callers (the ship's post-write render, CI's launch archive --verify) are covered today, but any new direct caller (itd-72's publish step) inherits no dirty gate. The render-path precheck also hands the documentation audit nil, so its doc-auditor row claims 'no .abcd/docs-lint.json' for a repository that has one. Remedy: a DirtyPolicy on PayloadRenderRequest whose zero value refuses a dirty tree, each live caller passing its policy explicitly, and a render-path doc-auditor row that says it was not measured rather than that the config is absent.

## Grounds

- pursued: a render request that states no policy refuses a dirty tree while the cut and the verified archive still render; a zero-value request rendering a dirty tree, or a cut refused by its own writes, would show it wrong
