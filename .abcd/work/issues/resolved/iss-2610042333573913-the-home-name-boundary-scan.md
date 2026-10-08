---
schema_version: 1
id: "iss-2610042333573913"
slug: "the-home-name-boundary-scan"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "noindex steps 3-4 landing (#815), abcd-50, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/abcdhome/boundary_test.go"
remedy: "Extend the scan to *_test.go files with a declared allowlist of the tests that plant or name the old home on purpose (the stop tests homestop_test.go and hooks_homestop_test.go, firstrun_home_test.go, the boundary tests' hostile sources, internal/fsutil's primitive tests with their own paths); every other test reaches the home through abcdhome.Path or abcdhome.Display, so a fixture follows the name when it changes; watched fail first on a planted filepath.Join(home, \".abcd\", ...) in a test file."
resolution: "The home-name boundary scan now holds test files too (TestNoTestBuildsTheHomeByHand) with a declared allowlist, testHomeSpellers, of the tests that plant or name the folder on purpose; every other test reaches the home through abcdhome.Path, Rel or Display. Failure/log messages and subtest names are prose and are exempt."
impact: internal
resolved_by:
  commit: "fc8802a45176410c7d4874a918878911dba4b6ff"
---

The home-name boundary scan (TestOnlyTheHomeResolverNamesTheHome, internal/abcdhome) walks non-test Go only, so a test can build abcd's home by hand, filepath.Join(home, ".abcd", ...) or a "~/.abcd/..." literal, and keep passing after the name changes. Observed 2026-10-04 merging main into the noindex rename (#815): guided-connect, interview and credential-guide tests merged after step 2 hand-built ~/.abcd paths; most failed once the home was renamed, and internal/core/credential/guide_test.go passed while seeding a connection into the old folder, which the code under test no longer reads, so it checked nothing.

## Grounds

- pursued: a test that builds the home by hand (filepath.Join(home, ".abcd", ...) or a ~/.abcd literal in a fixture or expected output) now fails the boundary test, watched on a planted file; it would be shown wrong by such a fixture passing outside the allowlist, or by an allowlisted file that no longer spells the home staying listed
