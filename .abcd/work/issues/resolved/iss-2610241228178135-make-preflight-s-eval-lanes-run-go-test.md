---
schema_version: 1
id: "iss-2610241228178135"
slug: "make-preflight-s-eval-lanes-run-go-test"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "make preflight on fix/verdict-confirm"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "Declare the eval lanes' budget: make smoke and make evals-cold-reading run go test -timeout $(EVAL_TIMEOUT) with EVAL_TIMEOUT ?= 20m, the workflow steps that run them pass a value below their job's timeout-minutes so a hang still dumps goroutines, and a lint test holds both."
resolution: "make smoke and make evals-cold-reading take -timeout $(EVAL_TIMEOUT), 20m by default; the CI and release steps set 8m under their jobs' ceilings; evallanebudget_test.go holds both"
impact: internal
---

make preflight's eval lanes run go test with no -timeout, so they inherit go test's ten-minute per-package default. On a loaded machine (load 21 to 31 on 10 cores, several sessions testing at once on 2026-10-24) the smoke lane's evals package ran past ten minutes with no test hung: TestEveryCommittedEntryFitsItsDeclaredWindow alone took 128 s, and the package was killed with 'panic: test timed out after 10m0s', failing a preflight whose change touched no eval. The unit and race lanes declare -timeout 20m and 40m; the eval lanes' ceiling is a default nobody sized.
