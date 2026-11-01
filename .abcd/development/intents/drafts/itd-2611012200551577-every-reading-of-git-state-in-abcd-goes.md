---
id: itd-2611012200551577
slug: every-reading-of-git-state-in-abcd-goes
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# Every reading of git state goes through internal/gitutil

## Press Release

> Every reading of git state in abcd goes through one package, internal/gitutil, which asks git in its pinned form (plumbing or porcelain formats with -z, --type for interpreted values, a fixed locale, configuration only through the protected command scope) and still cross-checks git's answer where a repository's configuration can steer it. Its only named exemptions, each with a test inside the package, are the three cases where git will not answer: git missing from PATH, a corrupt `.git`, and a checkout owned by another account before the person's trusted-roots declaration admits it, after which the root is handed to git as -c safe.directory for reads only, never on the commit or merge path, where the repository's own hooks and signing deliberately stay in force. Each answer that cannot change within one invocation (the root commit, the object format, the git version) is asked once and reused; anything that can change (HEAD, status, a ref) is asked again immediately before the act that relies on it. An AST test, in the shape of the one that keeps file locks inside fsutil, refuses a git execution that reads repository state, and a `.git` path opened or read, outside gitutil. Outside the rule are git writes (a commit, a config write, a move), the test-fixture builders in internal/gittest, `.git` names that are not reads (deny lists, a remote URL's suffix, path-safety gates such as fsutil.InsideGitDir, which gitutil itself imports), and reads that treat git's files as artefacts (a size, an mtime, a short shape check); each read names which tree it reads.

## Why This Matters

> _Why this matters to the user — replace before planning._

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
