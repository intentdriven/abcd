# Fix a finding in place, or file it? A state-of-the-art check

Date: 2026-10-27. Asked for by the product thinker, as the grounds of the remedy of
iss-2610271115000748 and of the discipline that states its rule. Recency was judged
against this date. Evidence tiers: **[EVIDENCE]** measured data, **[CONSENSUS]**
convergent practice across independent tools or teams, **[CONTESTED]** sources that
disagree, **[ANECDOTE]** a single practitioner's report.

## The question

When an automated pass (a linter, a consistency checker, a documentation reviewer,
a model-run reviewer) finds minor editing inconsistencies in a project's
documentation and design records, should each be fixed in place when it is
found, or filed as an issue for later? Where is the line between "fix it now"
and "file it, because it needs a decision"?

The occasion was the 2026-10-26 consistency pass over abcd's brief and intents
(`.abcd/work/reviews/2026-10-26-consistency/`). It filed 1,301 records from 1,307
findings. The triage that followed found 584 to be one-passage edits with a known
fix, 648 to need no change, and 69 to need a person's ruling.

## What the evidence says

1. **Check and merge before anything is fixed or filed.** No raw finding becomes an
   edit or an issue until it is confirmed real: duplicates merged by cause (with each
   copy's location kept), anything a record already settles dropped with the record
   named, and differences of level of detail (a summary against its detailed form)
   not counted as conflicts.
   - Google holds its review-time checks to under 10% "effective false positives",
     and switches a check off when more than 10% of reactions are "Not useful"
     (Sadowski et al., CACM 2018). **[EVIDENCE]** About half of the 2026-10-26
     pass's findings needed no change, roughly five times that bar.
   - Without filtering, a model flags "natural gaps between high-level documentation
     and detailed code" as inconsistencies; filtering cut the flag rate from 98% to
     14%, at 0.62 precision (DocPrism, Xu, Wahab, Holmes and Lemieux, arXiv 2025).
     **[EVIDENCE]**
   - Uber's uReview keeps generation, filtering, validation and deduplication as
     separate stages, with the lesson "Precision Is More Valuable than Volume"
     (2025). **[CONSENSUS]**
   - Duplicate reports carry extra information; merge them rather than discard them
     (Bettenburg et al., ICSM 2008). **[EVIDENCE]**
2. **A confirmed finding with a known fix is fixed in the pass that found it, not
   filed.**
   - Facebook's Infer, run as a batch job assigning issues to developers: "almost
     none of them were acted on". Moved into code review, the same analysis at the
     same false-positive rate reached a fix rate "over 70%". The authors put it down
     to the cost of switching context and of finding the right owner later
     (Distefano et al., CACM 2019). **[EVIDENCE]**, with a small batch sample.
   - Google's 2009 FindBugs Fixit filed 1,746 reports from 3,954 reviewed warnings;
     only 640 warnings were ever fixed, and a dashboard "saw little use" (Sadowski et
     al., CACM 2018). **[EVIDENCE]** Two companies independently reached the same
     result.
   - Google's documentation guide: "Change your documentation in the same CL as the
     code change". GitLab: "You don't need an issue to update the documentation."
     Fowler: fix it "right there and then", or note it and "come back the same day".
     **[CONSENSUS]**
3. **"Known fix" is the safe-fix test linters use.** A fix is safe when it changes no
   reader's understanding of what the project requires, or when a record the commit
   cites already decides the change. Choosing between two different statements of a
   rule with no record to say which wins is not safe: that is a ruling.
   - Ruff: "The meaning and intent of your code will be retained when applying safe
     fixes"; only safe fixes run by default. ESLint: avoid "any fixes that could
     change the runtime behavior", and offer a suggestion instead. Vale: an action is
     something "an agent can act on without deciding anything". Google's
     documentation guide: "First delete what you're certain is wrong, ignore what's
     unclear." **[CONSENSUS across tool designs]**
4. **Findings that share a cause are one finding, fixed as a group behind its
   detector.** A stale name in forty places is one finding with forty locations: add
   the rule that detects it first, clear every instance in one mechanical change,
   and let the gate stop new ones. abcd's own `fix-the-detector` principle already
   requires that order.
   - Google's Error Prone team fixes every existing instance before a check becomes
     a compiler error, split by owner and reviewed by that owner; of the developers
     who received those patches, 57% were happy, 41% neutral, 2% negative (CACM 2018).
     **[EVIDENCE]** The Go team found mechanical rewrites (gofmt -r, gofix)
     "relatively easy to prepare, execute, and review". **[CONSENSUS]**
5. **Editorial fixes go in a change of their own; tiny local ones may ride along.**
   Google's engineering practices ("It's usually best to do refactorings in a
   separate CL", while "Small cleanups … can be included") and Chromium ("CLs should
   only effect one type of change") agree; Herzig (MSR 2013) found up to 15% of bug
   fixes tangled with other changes. **[CONSENSUS]** Whether mixing slows review is
   **[CONTESTED]**: two preprints by one author (AlOmar et al., arXiv 2022 and 2024)
   point both ways.
6. **A safe fix outside the finder's area is proposed to its owner as a change, not
   filed.** Google sends generated fixes to owners as reviewable changes and offers a
   one-click "Please fix"; review assistants post suggestions the author applies.
   **[CONSENSUS]**
7. **Only real rulings are filed, and as questions, not backlog.** The commit that
   fixes a finding is its record; an issue records only a decision not yet made.
   Low-value reports cost maintainers real time: each security report to curl
   engages three or four people for half an hour to three hours, and in 2025 about
   20% were machine-generated noise (Stenberg, 2025). **[ANECDOTE]** "Backlogs are a
   big weight we don't need to carry" (Shape Up). **[practitioner opinion]**
8. **Each checker's precision is tracked, and a noisy checker is fixed first.** When
   more than about 10% of a checker's findings are not real, its prompt or rule is
   fixed before its output is acted on (Google's "Not useful" ratio, CACM 2018).
   **[EVIDENCE]**

## Contested points

- **Pre-existing findings.** Infer reports only problems a change introduces, which
  runs against the boy scout rule and abcd's `pre-existing-is-not-a-defence`. The two
  fit: suppression applies to findings shown to a change's author, while a dedicated
  consistency pass is checking the existing state.
- **Small nits.** Uber found readability nits rated poorly; Atlassian's RovoDev study
  (ICSE 2026) lists readability among the most-resolved comment types. Both are
  company self-reports.
- **Model reviewer reliability.** At Beko, 73.8% of bot comments were resolved and
  21.3% marked "Won't Fix", while mean pull-request closure time rose (Cihan et al.,
  arXiv 2024).

## Not worth adopting

- **Filing every finding as an issue**: filed or batched findings mostly go unfixed,
  and half of this pass's findings were not real.
- **Applying model-proposed edits in bulk with no safe/unsafe gate**: without the
  check of point 1, fixing in place turns false positives into churn, or "reconciles"
  a correct passage wrongly.
- **Auto-closing stale issues**: projects using a stale bot saw "a considerable
  decrease in their number of active contributors" (Khatoonabadi et al., arXiv 2023).
- **Discarding duplicates as noise**: merge them; they show how far a problem has
  spread.

## Where the line falls

On whether the fix needs a decision, not on its size. A finding becomes an edit in
the pass that found it when it is confirmed real against the current record, and its
fix preserves meaning or is already decided by a named record the commit cites.
Findings that share a cause are fixed as a group, the detector landing first. A safe
fix outside the finder's area is proposed to its owner as a change. A fix too big for
the change at hand goes in its own change, next. Only a finding whose fix would change
what a passage requires, with no record saying which version wins, goes to a person,
as a ruling. A finding that is not real is dropped, with the reason kept in the pass's
output so it is not raised again.

Against the product thinker's first statement of the lesson, the evidence adds four
things: check before fixing (the 50% that needed no change); fix a shared cause as a
group behind its detector, not one instance at a time; propose rather than file across
a boundary; and judge "too large" by what a reviewer can take in, which means a
separate change, not an issue. Both large studies (2018, 2019) predate model-run
agents and had humans making the fixes; agents make fixing cheap and move the
bottleneck to review, which strengthens points 3 to 5.

## Sources

- Distefano et al., Scaling Static Analyses at Facebook, CACM 2019:
  https://6826.csail.mit.edu/2020/papers/facebook-infer-cacm.pdf
- Sadowski et al., Lessons from Building Static Analysis Tools at Google, CACM 2018:
  https://6826.csail.mit.edu/2020/papers/google-analysis-cacm.pdf
- Ruff, fix safety: https://docs.astral.sh/ruff/linter/
- ESLint, custom rules: https://eslint.org/docs/latest/extend/custom-rules
- Vale, actions: https://docs.vale.sh/topics/actions
- markdownlint: https://github.com/DavidAnson/markdownlint
- The Go blog, gofmt: https://go.dev/blog/gofmt
- Fowler, Opportunistic Refactoring: https://www.martinfowler.com/bliki/OpportunisticRefactoring.html
- Google engineering practices, small changes: https://google.github.io/eng-practices/review/developer/small-cls.html
- Chromium, change tips: https://chromium.googlesource.com/chromium/src.git/+show/HEAD/docs/cl_tips.md
- Swett, Why the Boy Scout Rule is insufficient: https://www.codewithjason.com/?p=411
- Google documentation best practices: https://google.github.io/styleguide/docguide/best_practices.html
- GitLab documentation contribution guide: https://docs.gitlab.com/development/documentation/
- Herzig, The Impact of Tangled Code Changes, MSR 2013: https://www.microsoft.com/en-us/research/?p=165196
- AlOmar et al., refactoring reviews in OpenStack: https://arxiv.org/abs/2203.14404
- AlOmar, the Qt Refactor branch: https://arxiv.org/abs/2410.04678v2
- Khatoonabadi et al., stale bots: https://arxiv.org/abs/2305.18150
- Bettenburg et al., Duplicate Bug Reports Considered Harmful... Really?, ICSM 2008:
  https://people.csail.mit.edu/hunkim/images/b/b2/Papers_Bettenburg2008icsm.pdf
- Shape Up, Bets, Not Backlogs: https://basecamp.com/shapeup/2.1-chapter-07
- Stenberg, Death by a thousand slops: https://daniel.haxx.se/blog/2025/07/14/death-by-a-thousand-slops/
- Cihan et al., Automated Code Review In Practice: https://arxiv.org/html/2412.18531v2
- Tantithamthavorn et al., RovoDev Code Reviewer: https://arxiv.org/html/2601.01129v1
- Uber, uReview: https://www.uber.com/us/en/blog/ureview/
- Xu et al., DocPrism: https://arxiv.org/abs/2511.00215v1
- Google Research, resolving code review comments with ML:
  https://research.google/blog/resolving-code-review-comments-with-ml/
- GitHub Copilot code review: https://docs.github.com/en/copilot/how-tos/use-copilot-agents/request-a-code-review/use-code-review
- Anthropic's code-review plugin: https://github.com/anthropics/claude-code/blob/main/plugins/code-review/README.md
