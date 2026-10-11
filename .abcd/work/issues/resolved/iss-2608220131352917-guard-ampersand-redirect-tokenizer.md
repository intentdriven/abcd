---
schema_version: 1
id: "iss-2608220131352917"
slug: "guard-ampersand-redirect-tokenizer"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "bughunt-b-round-7"
found_at: "internal/core/guard/tokenize.go"
resolution: "guard tokenizer now recognises &> and &>> as a redirection before the list-operator split; the target is dropped and a preceding fd digit is kept"
impact: fix
resolved_by:
  commit: "eb99ce5"
---

guard tokenizer misses the ampersand redirection operators &> and &>> so a glued or spaced &>/dev/null splits the simple command and drops the dangerous flag out of command position, a silent allow on every blocker-tier entry