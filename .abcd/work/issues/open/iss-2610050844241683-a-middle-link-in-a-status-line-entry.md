---
schema_version: 1
id: "iss-2610050844241683"
slug: "a-middle-link-in-a-status-line-entry"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "re-review of the status-line consent guards (PR #830)"
origin: researcher-authored
production_mode: hand-written
found_at: "the status-line trust check in internal/core/ahoy"
remedy: "Walk the entry's link chain hop by hop with os.Readlink and judge every directory it passes through (not writable by group or other, owned by the person or root, outside the working tree), refusing a chain longer than a small bound."
---

The status-line trust check resolves a recorded entry with EvalSymlinks and judges the link's own directory and the final target's directory, but not the directory of a link in the middle of a longer chain. Example: the install record names ~/.local/bin/abcd, a link to /opt/homebrew/bin/abcd (a group-writable root:admin directory), itself a link to the real binary. Both judged directories pass, another admin account can repoint the middle link, and the status line then runs their binary on every refresh. Neither install shape abcd writes is a chain (a copy, or one link to the pinned plugin binary), so it is reachable only through a hand-edited install record that vouches for a hand-made chain.
