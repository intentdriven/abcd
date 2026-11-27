---
id: dcn-2611271216513537
slug: when-a-git-identity-value-user-name-user
date: 2026-11-27
---
2026-11-27 — When a git identity value (user.name, user.email, the remote's owner) or the account name taken from the home folder is not valid UTF-8, the redactor refuses the store write and says which setting to re-save, instead of guessing at the forms the text may hold. Grounds: two rounds of matching guessed forms each left a narrower case through, while before this branch such a value stopped every scan anyway, so refusing is no worse than main for anyone and never lets a name through. Home path spellings keep the received-form matching, which a review found complete. Reverse by matching the guessed forms again with a boundary rule that counts the replacement character as part of a word.
