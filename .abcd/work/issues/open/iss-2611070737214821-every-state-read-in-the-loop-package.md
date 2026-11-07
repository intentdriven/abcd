---
schema_version: 1
id: "iss-2611070737214821"
slug: "every-state-read-in-the-loop-package"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "follow-up list from a release cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop"
remedy: "Drop the rev-parse where ls-files can answer, cache per step, and cache the root commit; measure the loop package before and after."
---

Every state read in the loop package runs git rev-parse --is-inside-work-tree and then git ls-files -z (about 164,000 processes, 44% of the package's test time). Dropping the rev-parse alone saves about 22% if a plain ls-files failure can tell not-a-repository from git-refused; caching per step saves more. The root-commit rev-list (never changes) and repeated config and for-each-ref calls are the next leads.
