---
id: dcn-2611260133465269
slug: the-guard-counts-a-builtin-that-may-set
date: 2026-11-26
---
2026-11-26 — The guard counts a builtin that may set variables as setting one wherever it is handed a word the line does not fix followed by another word (print "$X" y, wait "$a" "$b"), and since the fixed-program reading now admits only a short list of commands that set nothing, such a builtin anywhere on the line leaves a variable-named program unread. A line that then runs a program named by a variable is held even where it is safe. Grounds: the unfixed word can be the option that names the variable the builtin sets, and the guard does not read which. To reverse: add the builtin to quietPrograms in internal/core/guard/fixedprogram.go with a check that every option word is fixed and names no variable, and pin the allowed lines in TestAnUnlistedCommandLeavesNoProgramFixed.
