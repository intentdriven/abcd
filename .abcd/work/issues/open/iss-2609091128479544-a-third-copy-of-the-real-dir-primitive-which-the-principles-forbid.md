---
schema_version: 1
id: "iss-2609091128479544"
slug: "a-third-copy-of-the-real-dir-primitive-which-the-principles-forbid"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "adversarial-review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/history/location.go"
---

internal/fsutil is the declared home for directory validation, and .abcd/development/principles/one-canonical-primitive.md names that primitive in its rule list: a second copy is a flagged consolidation target, a third is forbidden. There are three ensureRealDir implementations at this tip. internal/core/lifeboat/voyage.go:115 and internal/core/history/location.go:175 are near-identical (os.Mkdir at 0o700 followed by fsutil.IsRealDir, differing only in the error each returns), and the history one was added this cycle, so it is the copy that crossed the line the principle draws. internal/core/intent/lifecycle.go:824 is the divergent variant: an Lstat symlink check followed by os.MkdirAll at 0o755, whose own doc comment records that a symlinked ANCESTOR is not caught. The third copy is therefore not mere duplication but the weakening the principle predicts: the intent variant's guarantee is strictly weaker than the other two, the difference is invisible at its five call sites, and a hardening landed in either of the other homes reaches none of them. The principle's Promotion clause asks for a check over private redefinitions of the canonical names, and internal/fsutil/canonical_test.go's TestNoNonCanonicalAtomicWritePrimitives is exactly that check for writeFileAtomic, durableWrite, isRealDir and createExclusiveIn; ensureRealDir is simply absent from its name list, which is how three copies accumulated under an armed detector. Two accepted decision records now bless the divergence rather than flag it: adr-2609090717039680:56-60 presents the local helper as a virtue and adr-2609091014087993:71-73 commits a future worktree store to a fourth copy. Fix direction: one primitive in internal/fsutil that takes the create mode as a parameter, because 0o700 for a private store under the caller's home and 0o755 for a record directory in a shared worktree are both legitimate and must not be silently unified, plus a chain form that verifies every level it creates so the intent variant's documented ancestor gap closes rather than being carried forward; and ensureRealDir joins the canonical-primitive detector's name list so a fourth copy cannot land quietly.
