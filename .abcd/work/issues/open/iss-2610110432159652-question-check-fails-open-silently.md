---
schema_version: 1
id: "iss-2610110432159652"
slug: "question-check-fails-open-silently"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "abcd-60 drain run 2026-10-11, lane for iss-2610090642394550"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli"
remedy: "When the question check cannot find the repository root, still let the question through (fail open) but write one stderr line naming why (the root lookup failed, and its error) so the person and the transcript can see the check did not run; test it with a git that refuses rev-parse."
---

The guard's question check lets a question without abcd's label through without a word when git cannot name the repository root for a directory that looks like a repository. The hook writes nothing at all: an empty stdout, exit 0 and an empty stderr. The guard's own rule is to fail open, but loudly. On a loaded machine this silence was what TestModeMadeQuestionAbcdsSaysSo met on PR #858, before f1703a29f removed the 50ms root-lookup deadline. The silent branch is still there for any other reason the lookup fails.
