# Model-matrix runs: state of the art (2026-10-16)

Survey behind itd-2610161733316518 (one feature built across a model matrix) and the
principle [a model run is re-runnable, not reproducible](../../principles/a-model-run-is-re-runnable-not-reproducible.md).
Two passes: a sota-researcher survey, then an independent fit-challenge that checked each
claim against its source and against the tools installed on the development machine
(Apple M4 Max, 128 GB, 16 cores; llama.cpp b9860, mlx-lm 0.31.3, Ollama 0.14.2).
Evidence tiers: EVIDENCE (peer-reviewed or primary measurement), DOCS (vendor
documentation), ANECDOTE (single report or vendor claim).

## Repeats and power

- **Run-to-run spread exceeds the gap between pairings.** Ariño de la Rubia and Pafka,
  "Identical Runs, Different Results" (arXiv 2609.33812, Sep 2026): 584 trials across six
  harnesses and open-weight models. Within one model-and-harness pairing, spread exceeded
  the spread between pairings; three-run comparisons ranked the weaker pairing first in
  28–44% of resamples (Study 2). Separating the gaps they observed needed about **20–113
  runs per arm** (16·sd²/gap² at 80% power). Fit caveat: hosted open-weight endpoints with
  the upstream build unrecorded, not frontier models. EVIDENCE.
- **Error bars.** Miller, "Adding Error Bars to Evals" (arXiv 2411.00640): standard errors
  from the central limit theorem, clustered errors, paired per-question differences, and
  a power analysis before running. EVIDENCE.
- **Reliability.** pass^k, the chance all k tries succeed (Yao et al., τ-bench, arXiv
  2406.12045), measures what an autonomous loop needs; Inspect AI ships pass@k and pass^k
  reducers. EVIDENCE / DOCS.
- Attempts per task in practice: METR uses several attempts per task and bootstraps over
  families, tasks and attempts (METR modelling note, 2026-03-20). ANECDOTE for the count.

## The cell is model plus harness

- 2609.33812 found harness gains depend on the model, with rankings reversing between
  models. The authors flag this as exploratory, noticed after the data were in. EVIDENCE,
  caveated.
- SWE-bench's Bash Only track runs every model through one scaffold (mini-SWE-agent) and
  warns that its 1.x and 2.x results are not comparable. DOCS.
- Terminal-Bench 2.0 found model choice mattered more than scaffold. Contested against
  2609.33812; the design choice (harness version is part of the cell) stands either way.

## What served, not only what was asked

- OpenRouter's generation record (`GET /api/v1/generation?id=`) returns the served
  `model`, `provider_name`, `provider_responses` including fallback attempts, native token
  counts including reasoning and cached, `latency` and `total_cost`. No retention window
  is documented, so fetch at call time. DOCS.
- Provider pinning: `provider.order`, `allow_fallbacks` (default true),
  `require_parameters` (default false), `quantizations`, `data_collection`. Record the
  permanent `canonical_slug`; avoid `~` latest aliases. DOCS.
- Providers serving one model differ in tool-call accuracy beyond what benchmarks show
  (OpenRouter Exacto post); declared quantisation is self-reported (Kimi Vendor Verifier).
  ANECDOTE.
- Prompt-cache share drove a ~23× cost gap between harnesses (2609.33812). EVIDENCE.
- HAL (Kapoor et al., ICLR 2026) reports cost beside accuracy and found higher reasoning
  effort lowered accuracy in most runs, so effort is pinned per cell. EVIDENCE.

## The nondeterminism floor

- Thinking Machines (He, 2025): 1,000 temperature-zero requests to Qwen3-235B gave 80
  distinct completions; the cause is kernels that are not batch-invariant under varying
  load. Batch-invariant kernels made all 1,000 agree at roughly 1.6–2.1× cost. EVIDENCE.
- OpenAI's seed is best effort; OpenRouter says same-seed requests "should" match but
  determinism is not guaranteed for some models. DOCS.
- Anthropic engineering (Feb 2026, read via secondary coverage): sandbox resources alone
  moved Terminal-Bench 2.0 by six points; treat gaps under three points with suspicion and
  run on several days. EVIDENCE, secondary. Consequence: concurrency and resource limits
  belong in the cell's manifest.
- Inference, untested: a single-user local server at batch size one avoids the batch
  cause.

## Review

- Self-preference: Panickssery, Bowman and Feng (NeurIPS 2024) show models recognise their
  own text and recognition correlates with favouring it. Chen et al. (arXiv 2504.03846)
  find most self-preference on verifiable tasks is legitimate. This repository's own note
  ([cross-model review](2026-07-14-cross-model-review.md)) records it as weak or absent
  under genuine authorship. **Contested**: blinding is cheap insurance; scoring against
  hidden tests is the real control.
- Panels: PoLL (Verga et al., arXiv 2404.18796) favoured a cross-family panel over one
  judge on general text; the repository's later code-specific evidence finds consensus
  underperforms the best single judge. Reviewers are scored, never voted.
- CodeJudgeBench (arXiv 2507.10535): pairwise judging in both orders beats single scoring,
  and order changes verdicts. Pairwise over N copies costs O(N²) comparisons.

## Outcome validity and contamination

- Weak tests mis-estimate performance by up to 100% relative (ABC checklist, Zhu et al.,
  arXiv 2507.02825). EVIDENCE.
- Reject rule-breaking runs before ranking: in 2609.33812 runs that broke the task's rules
  held the top seven scores. EVIDENCE.
- Build a feature not yet in any public history; SWE-bench Pro resists contamination with
  copyleft and private repositories. CONSENSUS.

## Local serving on this machine (checked 2026-10-16)

- **llama-server b9860**: tool calling through the chat template; `--jinja` is on by
  default in this build. A q4 KV cache can degrade tool calling (llama.cpp
  function-calling docs). DOCS + local check.
- **mlx-lm 0.31.3 `mlx_lm.server`**: accepts `tools` and `seed`; a seed turns batching off.
  The risk is per-family tool parsers, as the [local-models note](2026-08-22-local-models-mlx-sota.md)
  says. Local check (`server.py`).
- **Ollama 0.14.2**: context length is server-wide (`OLLAMA_CONTEXT_LENGTH`, default
  4096) or per model in a Modelfile, not per OpenAI-compatible request; a long prompt is
  silently truncated at the default. Local check (`ollama serve --help`) + DOCS.
- **vllm-metal v0.28.0** (announced 2026-09-22): upstream vLLM's server and tool parsers
  on Metal; experimental. ANECDOTE.
- Pinning a local model: weight file sha256 or Ollama digest, Hugging Face revision,
  quantisation, context length, KV-cache type, runtime and version.
- Candidates: Qwen3.6-27B (dense) and Qwen3-Coder-Next (80B total, 3B active); card scores
  are vendor-reported on vendor scaffolds. ANECDOTE.

## Existing harnesses (path choice under sota-per-intent)

| Alternative | Maturity | Fit |
| --- | --- | --- |
| Harbor (Terminal-Bench 2.0 harness) | usable, fast-moving | Runs several coding agents with n attempts in containers; Python + Docker; runs host agents on API keys, which clashes with host-only frontier models and the credential store. |
| Inspect AI + inspect_swe | mature / usable | Epochs, pass@k reducers, eval logs pin model and settings; Python. Its eval spec is the closest manifest template. |
| HAL harness | usable | Cost-aware; leaderboard paused. |
| SWE-bench harness, mini-SWE-agent | de facto / mature | Test grading in Docker; single agent, not a multi-role loop. |
| OpenHands eval, aider benchmark, promptfoo | usable–mature | Each tied to its own agent or to prompt-level evals. |
| LiteLLM proxy | mature | One recorder for every provider, but holds its own keys, against the one-reader credential rule. |

Verdict: path 2, a native run on abcd's own loop, with a real seam: one trial interface
(frozen commit, intent, cell configuration → trial record) whose record maps field by
field onto a Harbor trial result or an Inspect sample. Path 1 would need a Python and
Docker dependency and API-key credentials for host models.

## Not adopted

- Temperature zero plus a seed as a reproducibility guarantee (refuted above).
- A single judge, or reviewers who are told the author.
- Ranking models from one feature at a few runs each.
- OpenTelemetry GenAI conventions as the manifest format: the spec has moved to its own
  repository with no release; borrow attribute names only.

## Addendum 2026-10-16: doc writing judged by people, and doc fidelity compared

Added when the product thinker widened the run to any task (code, doc writing, doc
fidelity). Second sota-researcher pass; recency checked against 2026-10-16.

### People judging copies of a doc

- **Rank, do not score alone.** Ranking-based methods are more reliable than single
  rating scales (Novikova et al., RankME, NAACL 2018; Kiritchenko and Mohammad, best-worst
  scaling, ACL 2017); van der Lee et al. (INLG 2019) recommend multi-item scales or
  ranking. HELM Instruct's 1–5 rubric reached only Pearson 0.48–0.72 between evaluator
  pairs. EVIDENCE.
- **Randomise the order per judge and calibrate first** on one good and one bad example;
  shorter sessions limit fatigue (van der Lee 2019). CONSENSUS.
- **Judge accuracy against the code, not how it reads.** Untrained raters commented on
  form about twice as often as content and agreed at Krippendorff's α ≈ 0 (Clark et al.,
  ACL 2021); training with examples moved attention to content. EVIDENCE.
- **Record length beside every verdict.** Length was the dominant style factor in Arena's
  human votes; controlling for it moved models by up to twelve ranks (LMSYS, 2024-08-28).
  EVIDENCE, observational.
- **Blinding is partial.** A classifier named which of five models wrote a text with 97.1%
  accuracy, surviving rewriting (Sun et al., ICML 2025). Render every copy through one
  template, strip self-references, ask each judge to guess the model as a leak check.
  EVIDENCE.
- **Combine with Bradley–Terry, not a vote or Elo.** Arena moved to Bradley–Terry for
  estimation (Chiang et al., 2024); Elo depends on order (Boubdir et al.). EVIDENCE.
- **Report Krippendorff's α with percentage agreement**; ≥0.800 reliable, 0.667–0.800
  tentative; NLG studies typically reach 0.3–0.5. CONSENSUS.
- **Small designs are weak.** Median NLG study: four annotators, 100 items (van der Lee);
  common designs are underpowered for small differences (Card et al., EMNLP 2020). One
  page judged by three people decides which copy to ship, not which model is better.
  EVIDENCE.

### Comparing the findings of review copies

- **Repeat before calling a finding unique.** In SWR-Bench (FSE 2026), five runs of one
  model agreed on fewer findings than different models did; a single-run singleton is
  mostly noise. EVIDENCE.
- **Match on exact keys first; a model only for leftovers, validated.** A preprint matches
  on file, ±5 lines and type, sending only ambiguous pairs to a model (Kumar et al., arXiv
  2606.15689). SWR-Bench validated its model matcher (κ 0.71–0.87); c-CRAB (arXiv
  2603.23448) rejects model judges for randomness. Validate any matcher on about thirty
  human-labelled pairs. CONTESTED.
- **Adjudicate blind and report precision, unique valid findings, recall against the
  adjudicated union, and a pairwise Jaccard matrix.** A union raises recall but not
  precision (c-CRAB, Kumar et al.); recall against the union overstates true recall.
  EVIDENCE.
- **Show it as an UpSet-style table** (one row per matched finding, one column per model)
  with a Jaccard heatmap; Venn diagrams only up to three sets (upset.app). CONSENSUS.

### Local fit

- abcd's consistency findings already carry a stable key (`findingkey.Derive`: class, both
  ends' paths and quotes, whitespace collapsed); two models agree on a key only when they
  quote the same text. Doc-fidelity findings carry no key; their join is document,
  chapter and quoted sentence, so matching also needs line or span overlap.
- The doc-fidelity receipt keeps one verdict per commit, and consistency ingest files
  issues with no dry run: a matrix run stores each copy's findings in its own record and
  files nothing.
- The release-changelog composer is the natural first doc-writing task: fixed input from
  a cut, and the binary already checks every citation.

Not adopted: Arena's adaptive sampling and style regression (thousands of votes needed);
rewriting copies with a model to blind them (the fingerprint survives and the content
changes); crowd raters for doc quality.
