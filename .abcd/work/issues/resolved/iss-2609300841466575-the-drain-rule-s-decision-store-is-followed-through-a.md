---
schema_version: 1
id: "iss-2609300841466575"
slug: "the-drain-rule-s-decision-store-is-followed-through-a"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
remedy: "Before reading the store, examine each directory from the checkout down to the store with os.Root.Lstat and refuse (ErrUnreadable, exit 2, 'is not a regular directory') any that is a link or not a directory, whether it points inside or out of the checkout, matching readRecord's refusal of a linked record; update Load's doc comment and the brief's trust boundary."
resolution: "drainrule.Load examines each directory from the checkout down to the store without following it and refuses a link with ErrUnreadable ('is not a regular directory', exit 2), inside the checkout or out, as a linked record is refused."
impact: fix
resolved_by:
  commit: "f2c11d36c"
---

The drain rule's decision store is followed through a symlink that resolves inside the checkout, while a record that is a symlink is refused wherever it points. drainrule.Load reads .abcd/development/decisions/adrs through os.Root, which follows a link staying inside the root, so a linked store (or a linked directory above it) loads a rule from a path the checkout does not commit as the store; a store linked out of the checkout was refused only by os.Root's escape check. Named as INFO by reverify-drainOwnRule on b93f4cdd3.

## Grounds

- pursued: a store linked inside the checkout, out of it, or below a linked parent refuses with ErrUnreadable (TestASymlinkedStoreIsRefusedWhereverItPoints, TestEveryRefusalOfTheRuleExitsTwo); a rule loading through any linked store would show it wrong
