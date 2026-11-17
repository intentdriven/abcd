---
schema_version: 1
id: "iss-2611141504118241"
slug: "a-reviewer-s-passing-verdict-is-refused"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "a drive's records round"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/validate.go"
remedy: "In the verdict reader (internal/core/implement/loop/validate.go, the rule that reports which commit a verdict names), take the commit a verdict names from the first full commit id after the verdict word on the verdict line, and ignore later ids on that line; and have the review brief say that the verdict line holds the verdict word and the judged commit and nothing else. Proven by a validate test with a verdict line that names the judged commit and then a second commit (accepted), and one that names only another commit (still refused)."
resolution: "The verdict reader takes the first full object name or digest after the verdict word on the verdict line as what the verdict judges and does not read the rest of that line; a line naming only another commit, or another commit first, is refused, and a line with no full name reads as before. The loop's reviewer brief says the verdict line holds the verdict word and the judged name and nothing else."
impact: fix
resolved_by:
  commit: "e8112886614b3a6a12ea5299e9a6c3026a02d80d"
---

A reviewer's passing verdict is refused when its verdict line names the judged commit first and then mentions another commit in the same sentence. In a landing's records round the reviewer wrote the pass word, the full id of the records commit under review, and went on to say that the record's resolved_by matches the lane's own commit, giving that commit's full id. The validator answered that the verdict names the second commit and that the review judges the first, refused the return, and the launch was repeated; the reviewer wrote the same shape again. A records commit always cites the commit it resolves with, so a careful reviewer is likely to name both.

## Grounds

- pursued: a records round's pass that names the judged commit and then the resolving commit is accepted; a refusal of that shape would show it wrong
