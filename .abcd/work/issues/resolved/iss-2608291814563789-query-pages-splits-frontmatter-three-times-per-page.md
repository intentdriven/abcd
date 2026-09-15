---
schema_version: 1
id: "iss-2608291814563789"
slug: "query-pages-splits-frontmatter-three-times-per-page"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "internal/core/memory/ask.go"
resolution: "parsePage splits and parses each page once and QueryPages threads it into PageInfo, the body and the source block; on a synthetic 500-page store one question drops from 11.7 ms and 54.1k allocs to 9.2 ms and 32.6k allocs (BenchmarkQueryPages)"
impact: internal
---

ultra-v0.6.8 C9: QueryPages in internal/core/memory/ask.go splits every page's frontmatter three times — pageInfoFrom, pageBody and pageSourceBlock each call splitFileFrontmatter (a full normaliseNewlines plus strings.Split of up to maxMemoryPageBytes), and two of them re-run parseFrontmatter on the same region. abcd memory ask does this over the whole store per question. Fix: split and parse once per page and thread (region, body) into the three consumers; measure before and after on a synthetic store.
