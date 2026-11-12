---
schema_version: 1
id: "iss-2611040555499163"
slug: "the-question-limits-and-abcd-guard"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: "CHANGELOG.md"
remedy: "Have the delivering change carry a record; name these in the next cut's changelog."
refines: [iss-2609012313465609]
resolution: "A repository sets three limits on abcd's own questions in .abcd/config.json: question.rows (24 to 60), question.label_words (4 to 8) and question.description_sentences (2 to 4); a person's own ~/.abcd.noindex/config.json may set each lower, never higher. The rows default rose from 24 to 32, so a team on 24-row screens sets question.rows to 24. A value outside its bounds is refused, naming the key, the value, the bounds and the file, and the hook then refuses every question of abcd's until the file is fixed. abcd guard question checks a drafted question before it is asked, read-only, with the hook's own decision: it reports every finding, each tab's rows, and the limits in force with the file that set each, and exits 0 with no finding, 1 on any finding and 2 on an unreadable draft, a refused limit or a store it could not read. Both shipped in v0.16.1 with no changelog line (the work of itd-2610301914350541 and itd-2610230756259215, each still planned), so this record names them in the next cut."
impact: additive
resolved_by:
  commit: "677771841e4084c969b7817d0ffc829d149b493b"
---

The question.* limits and abcd guard question are new since v0.16.0, and default rows rose from 24 to 32, with no line in the v0.16.1 changelog, because no record in the cut covers them.

## Grounds

- pursued: the next cut's Added section names the question limits and abcd guard question from this record; a cut whose changelog names neither would show it wrong
