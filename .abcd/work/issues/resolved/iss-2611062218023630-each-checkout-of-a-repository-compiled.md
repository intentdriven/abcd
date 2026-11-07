---
schema_version: 1
id: "iss-2611062218023630"
slug: "each-checkout-of-a-repository-compiled"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "build cache growth review"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/abcd.sh"
remedy: "Build under -trimpath in scripts/abcd.sh and the commit hooks, export it in GOFLAGS from the Makefile and from ci.yml's workflow env so preflight and CI test one form, and admit -trimpath as a narrow test's flag."
resolution: "scripts/abcd.sh and the commit hooks build under -trimpath, the Makefile and ci.yml's workflow env set it in GOFLAGS, and a narrow test may carry it, so checkouts share one build cache and preflight tests the form CI tests."
impact: internal
resolved_by:
  commit: "8abba6bbb81d777fde7eb77eab909118e1e9b982"
---

Each checkout of a repository compiled its own copy of every package, because go puts a package's directory in its compile key unless the build passes -trimpath, and scripts/abcd.sh, the commit hooks, the Makefile's build, vet and gate runs and the commands the load gate runs did not pass it. Only the test reuse stage did. On one development machine the build cache held 217.7 GB of compiled packages across about 99 worktrees and 76 review copies. Measured with a private cache on the declared toolchain, go 1.26.10, four jobs, on a loaded machine: a second checkout's build of ./cmd/abcd added 738 files and 170.5 MiB in 6.3 s without the flag, and 245 files and 1.9 MiB in 1.0 s with it, and the two checkouts linked one identical binary only with it; a cold build was about 490 MiB either way. A second checkout's go vet ./cmd/abcd added 247 files and 1.0 MiB without the flag, and nothing with it.
