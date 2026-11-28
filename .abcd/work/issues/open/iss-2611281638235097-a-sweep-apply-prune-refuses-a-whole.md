---
schema_version: 1
id: "iss-2611281638235097"
slug: "a-sweep-apply-prune-refuses-a-whole"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "sweep --apply on the person's word, 2026-11-28"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core"
remedy: "Have the notes archive skip known regenerable caches (__pycache__/ directories, *.pyc, .pytest_cache/, .mypy_cache/, node_modules/.cache/) under a worktree's local tier, naming each skipped path in the prune's report, and keep refusing every other non-text file; test that a worktree whose scratch holds a .pyc beside its .py is archived with the .py and pruned. Grounds: git's own default ignore templates and Python's PEP 3147, which defines __pycache__ as a regenerable cache."
---

A sweep --apply prune refuses a whole worktree when its local notes hold a binary cache file: archiving lane-ab-followups failed on scratch/__pycache__/wrap.cpython-314.pyc as not text, which the redaction pass cannot vouch for, so sweep exited 2 and kept the worktree. A Python bytecode cache is regenerated from its source and holds nothing the notes archive needs.
