# Spot-checking what agents changed with no person reading it: a state-of-the-art check

Date: 2026-10-30. Asked for by the product thinker as the evidence for
itd-2610301750162657. Recency was judged against this date. Evidence tiers:
**A** peer-reviewed work or a published standard, **B** a rigorous industry or
empirical study, **C** a practitioner report. Every source linked below was
opened on that date. Not opened, so not relied on: the original Hanley and
Lippman-Hand paper (JAMA 1983; cited through Newman's 1995 restatement), AICPA
AU-C 530 (its international twin ISA 530 is used instead), the text of ISO
2859-1 and ANSI Z1.4, the continuous-auditing literature, and papers on
monitoring learning-to-defer systems.

## The question

A repository lets coding agents land changes that no person reads: an
unattended drain fixes issues whose recorded fields say they need no decision,
and pull requests that automated reviewers and a merge queue approve can merge
with no person asked. 488 changes landed on abcd's own default branch in the 30 days to
2026-10-30 (merge commits on its first-parent line). Should a person read a small sample of that stream afterwards, and
if so, how should the sample be drawn, sized and acted on?

## What the evidence says

1. **Draw the sample from a seed the agents cannot touch, and give every
   unattended change a chance of selection** (tier A). Audit sampling requires
   that "each sampling unit in the population has a chance of selection" (ISA
   530, para 8), and defines statistical sampling as random selection plus
   probability-based evaluation (para 5(g)),
   <https://www.ifac.org/system/files/publications/files/A028%202012%20IAASB%20Handbook%20ISA%20530.pdf>.
   When the system under test decides which cases get an outcome, evaluation
   happens on "a difficult-to-interpret non-random sample of the population"
   (Lakkaraju, Kleinberg, Leskovec, Ludwig and Mullainathan, KDD 2017,
   <https://www.cs.cornell.edu/home/kleinber/kdd17-selective.pdf>). A drain
   decides what needs no decision and what is handed back, so the handed-back
   group is sampled too, or the audit sees only what the agent chose to
   release.
2. **One finding is a defect in the rule, never "within tolerance"** (tier A
   for the standard; C for the zero-acceptance plans). ISA 530 requires the
   auditor to "investigate the nature and cause of any deviations" (para 12),
   and allows a deviation to be written off as an anomaly only with "a high
   degree of certainty" from further procedures (para 13). Zero-acceptance
   (c=0) sampling plans accept only on zero defects (Squeglia, ASQ Quality
   Press, <https://asq.org/quality-press/display-item?item=H1607>; book text
   not opened).
3. **A small clean sample bounds a rate; it does not certify one** (tier A).
   Zero events in n trials gives a 95% upper bound of about 3/n (Newman, a letter in JAMA
   1995, restating Hanley and Lippman-Hand,
   <https://jamanetwork.com/journals/jama/article-abstract/389714>). So 30
   clean reads rule out a failure rate above about 10%, 60 above 5%, 100 above
   3% and 300 above 1%. At the one published revert rate for agent pull
   requests (2.66%, below), a sample of 30 would contain such a change less
   than once on average: a small sample finds a rule that fails, not a rate.
4. **Stratify by what happened to the change, not by which agent wrote it**
   (tier B). Of rejected agent pull requests, 38% were closed with no
   meaningful human interaction and only 2% failed at the agent level (Ehsani
   et al., MSR 2026, <https://arxiv.org/html/2601.15195>); per-agent
   differences in post-merge quality "mostly disappear once normalized by code
   churn" (Cynthia et al., MSR 2026, <https://arxiv.org/html/2601.20109v1>).
   The written rule acts on disposition, so disposition is the stratum; the
   group that merged with no person looking is where reading matters most.
5. **The reader sees the change before the automated verdict, and is not the
   rule's author** (tiers A and B). Automation bias "cannot be prevented by
   training or instructions", and complacency appears in naive and expert
   users alike (Parasuraman and Manzey, *Human Factors* 2010,
   <https://pubmed.ncbi.nlm.nih.gov/21077562/>). Higher code complexity
   significantly increased reviewers' acceptance of incorrect suggestions
   (n=385; Singh et al., FSE 2026, <https://susi.usi.ch/usi/documents/336576>).
   An industrial study reports the mechanism in one practitioner's words: "if
   any other issue exists, the bot would have written it" (Cihan et al.,
   <https://arxiv.org/html/2412.18531>). The NIST AI Risk Management Framework
   asks for assessors "who did not serve as front-line developers" (MEASURE
   1.3, <https://nvlpubs.nist.gov/nistpubs/ai/NIST.AI.100-1.pdf>).
6. **Automated review approval is weak evidence on its own** (tier B). On
   real preventable defects with pull-request context, the highest-recall
   code-review agent configuration reached 32.8% recall at 5.1% precision, and
   low-severity defects had the lowest recall (CR-Bench,
   <https://arxiv.org/html/2603.11078v1>). Across 278,790 review
   conversations, AI agents' suggestions were adopted at a significantly lower
   rate than people's, and over half of those not adopted were incorrect or
   fixed another way (Zhong et al., <https://arxiv.org/abs/2603.15911>).
7. **Look first for scope and side effects** (tier B). Of 500 reverting
   commits on agent pull requests, 22.3% undid "unintended side effects and
   overengineering", 22.1% functional errors, 17.7% code quality and 12.5%
   dependency problems; 2.66% of agent pull requests carried a revert (from
   0.7% to 7.6% by agent) (Oukay et al., MSR 2026 mining challenge,
   <https://2026.msrconf.org/details/msr-2026-mining-challenge/27/>).
8. **The output of a finding is a change to the rule, made and reviewed by a
   person** (tiers A and C). "An unreviewed postmortem might as well never have
   existed" (Google, *Site Reliability Engineering*, ch. 15,
   <https://sre.google/sre-book/postmortem-culture/>). NIST AI RMF MANAGE 4.1
   and 4.2 call for post-deployment monitoring with override and change
   management, and for measurable continual improvement. A person able "to
   disregard, override or reverse the output" is what EU AI Act Article 14(4)(d)
   requires of human oversight (<https://artificialintelligenceact.eu/article/14/>),
   and oversight that is not evidenced can "legitimize" faulty systems (Green,
   2022, <https://arxiv.org/abs/2109.05067>).
9. **Read less after clean cycles, more after a finding** (tier A for the
   handbook). Skip-lot sampling's average sample number "is smaller than the
   ASN of the reference sampling plan" (NIST/SEMATECH e-Handbook 6.2.7,
   <https://www.itl.nist.gov/div898/handbook/pmc/section2/pmc27.htm>), and
   acceptance plans are "one-shot deals" that "test short-run effects" (6.2.1,
   <https://www.itl.nist.gov/div898/handbook/pmc/section2/pmc21.htm>). ISO
   2859-1's own switching rules were not opened.

## Not worth adopting

- **Acceptance plans that tolerate defects (AQL, c greater than 0).** They
  accept a level of defects; the spot-check exists to repair the rule, so a
  tolerance defeats it.
- **A classifier that retunes the rule from the findings.** It removes the one
  place a person rules (EU AI Act Article 14(4)(d); Green 2022), and the
  drain's own ADR (adr-2609291342092738) already rejected a model sorter.
- **Sizing the sample to estimate a defect rate.** It needs hundreds of reads
  per group, which would crowd out careful reading by one person, the binding
  constraint in point 5.
- **Planted "gold" changes to grade the reviewers.** Crowdsourcing quality
  control injects items "whose answers are known... a priori", but making them
  is "very difficult and costly" (Daniel et al., *ACM Computing Surveys* 2018,
  <https://ar5iv.arxiv.org/html/1801.02546>); point 6 already measures the
  reviewers from live items.
- **Stratifying by model vendor** (point 4).

## Background, not evidence

Two pieces describe the same strain from the managers' side, and are kept as
context only: Fosslien and West Duffy, "Managers Are Struggling to Keep Up with
the AI Productivity Boom", *Harvard Business Review*, 25 May 2026 (reprint
H096N7, <https://store.hbr.org/product/managers-are-struggling-to-keep-up-with-the-ai-productivity-boom/H096N7>), on an "always-on review environment" and on using AI to pick which
work merits careful review; and TechCrunch's report of a Nielsen survey of
1,700 managers for Monday.com (21 October 2026,
<https://techcrunch.com/2026/10/21/ai-promises-productivity-but-for-managers-its-mostly-a-headache-for-now-says-new-survey/>),
in which nearly half spend more time "reviewing and guiding" and 46% worry about losing
control of agents' actions.
