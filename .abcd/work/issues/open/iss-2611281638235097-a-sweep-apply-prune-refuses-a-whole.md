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
found_at: "internal/core/ahoy/worktree/notes.go"
remedy: "Leave out of the notes archive only a file that is both named *.pyc under a __pycache__/ directory and not text (the same name-and-contents test iss-2611041759314115 settled for abcd's own outputs, so a note a person left under a cache-shaped name is still archived), naming each one left out in the prune's report through the same redaction as the other refused names (named, notes.go); keep refusing every other non-text file; test that a worktree whose scratch holds a .pyc beside its .py is archived with the .py and pruned, and that a text file named *.pyc under __pycache__/ is archived. Grounds: Python's PEP 3147, which defines __pycache__/*.pyc as a cache regenerated from its source; precedents iss-2611041759314115 and iss-2611181412255215."
---

A sweep --apply prune refuses a whole worktree when its local notes hold a binary cache file: archiving lane-ab-followups failed on scratch/__pycache__/wrap.cpython-314.pyc as not text, which the redaction pass cannot vouch for, so sweep exited 2 and kept the worktree. A Python bytecode cache is regenerated from its source and holds nothing the notes archive needs.
