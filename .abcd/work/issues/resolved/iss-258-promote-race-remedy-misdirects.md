---
schema_version: 1
id: "iss-258"
slug: "promote-race-remedy-misdirects"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "spc-24 build, ruthless-reviewer note"
found_at: "internal/core/capture/promote.go"
resolution: "The already-promoted refusal wraps ErrAlreadyPromoted, and a promotion whose stamp is refused that way (it lost the race to a concurrent promotion) is told which intent won and to delete its duplicate draft, instead of the link remedy that would be refused."
impact: fix
resolved_by:
  commit: "14ab2e07"
---

capture promote's post-mint stamp-failure remedy is attached unconditionally, so a promote that loses a race against a concurrent promote of the same issue emits advice that will refuse: B mints itd-Y, fails the under-lock re-check because A stamped itd-X, and B's error still says 'complete the link with capture promote iss-N --intent itd-Y' — running that hits the already-promoted refusal, and the duplicate draft itd-Y must be deleted by hand. The wrapped error does carry the true itd-X fact. Accepted residue class in spc-24 (no cross-store lock); a smarter remedy would special-case the already-promoted stamp error and say 'delete the duplicate draft' instead.

## Grounds

- pursued: the losing side of a promote race gets advice that works; a lost-race report still naming the --intent link would show it wrong
