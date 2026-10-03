# Host question layout: calibration at 80 by 24 (2026-10-03)

The host evidence for step 5 of spc-2610030944505997 (asking and layout): how Claude Code draws a question from its question tool
in a narrow and a wide terminal window, and the figures the question check's
row estimate takes from it (`HostTextColumns`, `HostChromeRows` and
`HostOptionColumns` in `question.Default`). The product thinker took the
screenshots on 2026-10-03 in Claude Code 2.1.288, in macOS Terminal. The images
are kept in the local tier, under `.abcd/.work.local/receipts/`, and are not
committed: this note describes them in words (the spec's open question 6,
decided (c)). The figures the tests read are the ones recorded here.

## What each screenshot shows

### 1. 80 by 24, with previews (`host-question-80x24-previews`)

A question headed "Product Q1" with three options, each carrying a side
preview. With previews present, the host shows the option labels only: every
option's description is hidden. Beside the labels it draws one preview box, for
the focused option. The box showed one line ("Gain: the folder stays small.")
and a marker reading "2 lines hidden".

The 24 rows, top to bottom: the chip (1), a blank (1), the question text in
lines and blank lines (10), a blank (1), the options beside the preview box
(4), a blank (1), the "Notes" row (1), a blank (1), the separator (1), "Chat
about this" (1), a blank (1), and the footer of key hints (1).

### 2. 80 by 24, descriptions and no previews (`host-question-80x24-descriptions`)

A question headed "Product Q2" with three options, each with a description, and
no previews. It shows two measures:

- **The question text.** A line wrapped after 72 characters, because the next
  word, eight characters long, did not fit. The usable text measure is
  therefore 76 columns, the provisional value the check already carried.
- **An option's description.** It sits at an indent of 5 columns, and it wrapped
  after 74 characters.

Without previews the host adds a free-text row ("Type something.") after the
options. The 24 rows: the chip (1), a blank (1), the question text (9), a blank
(1), the options with their descriptions (7: each label one row, then its
description's rows), "Type something." (1), the separator (1), "Chat about
this" (1), a blank (1), and the footer (1).

So without previews the frame costs 8 rows: the chip, the blank under it, the
blank before the options, the free-text row, the separator, the chat row, the
blank under it, and the footer.

### 3. 160 by 24, with previews (`host-question-160x24-previews`)

At 160 columns the side preview box stays about 45 columns wide. It showed one
line and "5 lines hidden". Height cuts it, not width: a wider window does not
show more of the preview.

### 4. 160 by 24, the orchestrator's own question (`host-question-160x24-preview-cut-own-question`)

A real question from the run that day, with previews, cut the same way: one
line of the focused option's preview shown and the rest hidden.

### 5. A second, independent observation

A sibling session saw the same layout at about 14:35 the same day (its images
were not kept). In a window about 200 columns wide it showed the preview in
full, but still no option descriptions. At 80 by 24, a preview of 9 to 10 lines
was cut to its first line, with "10 lines hidden".

## What follows

- **The measured figures.** `HostTextColumns` is 76, the question text's
  measure. `HostOptionColumns` is 74, the measure of an option's label and
  description at their indent. `HostChromeRows` is 8, the frame without
  previews, the chip and the free-text row included. The row estimate of
  screenshot 2's shape equals the 24 rows the host drew
  (`TestRowEstimateMatchesTheHostAt80By24`).
- **No side previews.** With a preview, the host hides what every option means
  and cuts the preview itself to the rows left, at any width. On this evidence
  the product thinker decided "No side previews" (the layout intent's decision
  20). abcd's questions put each option's meaning, gain and cost in its
  description, and the question check refuses a preview (its rule 14).
  Criterion A5 is reworded to match. The preview layout is therefore not
  estimated.
