---
schema_version: 1
id: "iss-2611252111095541"
slug: "review-all-user-facing-texts"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "manual build of 25 November"
origin: researcher-authored
production_mode: hand-written
found_at: "the text every abcd command prints"
remedy: "List every text the command line prints, by command and by the moment it is met; cut each to what its reader needs then to act or decide, moving a reason worth keeping to abcd explain or a page; install, update and the commonest commands first, one command family per change; then hold a length or a wording rule by a check, so it does not grow back."
---

What the abcd command line prints is too often an explanation where a word would do. The product thinker asked on 2026-11-25 for a thorough review of it, so that each text says what its reader needs and nothing more; their example is "(verified against the release checksums)" where "(verified)" would do. The focus is what a person sees when they type a command: help, results, warnings, refusals and their hints, and the install and update procedure, with the notices a session's start and stop print. A reader who wants the reason can be told where it is (`abcd explain`, a flag, a page); the line itself does not carry it. Other user-facing texts (the README, the pages under docs/ and the plugin's command pages) are in the wider request and come after.

A second example of theirs, reproduced on this tree: `abcd version` is refused in three lines that list all 45 commands, explain that version is a root flag, describe the board's last line and the update check, and point to `abcd explain`. "Did you mean `--version`?" is all its reader needs. A mistyped command (`captre`) gets the same list and no suggestion, so the shape is every unknown command's and not this one's.

Measured the same day: the help of `abcd mode` holds 637 words, of `abcd capture` 601, of `abcd update` 419, of `abcd ahoy` 371. Setting abcd up is already covered by itd-2610240823055889, which this does not replace.
