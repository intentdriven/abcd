package main

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// repoRoot is the module root as this test's working directory reaches it.
var repoRoot = filepath.Join("..", "..")

// moduleLister lists the module's packages as the job does, once per test.
func modulePackages(t *testing.T) []string {
	t.Helper()
	pkgs, err := goList(repoRoot)
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	if len(pkgs) < 50 {
		t.Fatalf("go list named %d packages; the module holds far more, so the listing lost its shape", len(pkgs))
	}
	return pkgs
}

// checkPartition fails unless shards hold every package exactly once, with no
// shard empty: a package in no shard is tested nowhere and nothing says so, and
// an empty shard's `go test` would test the working directory instead.
func checkPartition(t *testing.T, pkgs []string, shards [][]string) {
	t.Helper()
	seen := map[string]int{}
	for i, shard := range shards {
		if len(shard) == 0 {
			t.Errorf("shard %d of %d is empty", i+1, len(shards))
		}
		for _, p := range shard {
			seen[p]++
		}
	}
	for _, p := range pkgs {
		if seen[p] != 1 {
			t.Errorf("%s is in %d shards; want exactly one", p, seen[p])
		}
	}
	for p := range seen {
		if !slices.Contains(pkgs, p) {
			t.Errorf("%s is in a shard but not in the module", p)
		}
	}
}

// TestShardsPartitionTheModule: over the module as `go list ./...` names it, at
// every count a workflow could carry, the shards' union is the module's packages
// and no package is in two.
func TestShardsPartitionTheModule(t *testing.T) {
	pkgs := modulePackages(t)
	for of := 1; of <= 6; of++ {
		t.Run(fmt.Sprintf("of=%d", of), func(t *testing.T) {
			shards := assign(pkgs, of)
			if len(shards) != of {
				t.Fatalf("assign gave %d shards; want %d", len(shards), of)
			}
			checkPartition(t, pkgs, shards)
		})
	}
}

// TestAPackageAddedLaterLandsInExactlyOneShard is why the assignment reads the
// module rather than a list someone keeps: a package the weight table has never
// heard of is still tested, once.
func TestAPackageAddedLaterLandsInExactlyOneShard(t *testing.T) {
	const added = modulePath + "internal/core/addedafterthetablewaswritten"
	pkgs := append(modulePackages(t), added)
	checkPartition(t, pkgs, assign(pkgs, 3))
}

// TestAssignmentIsDeterministic: every shard computes the assignment on its own
// runner, so all of them must reach the same one, whatever order they read the
// module in. A second reading naming one package twice changes nothing either.
func TestAssignmentIsDeterministic(t *testing.T) {
	pkgs := modulePackages(t)
	want := assign(pkgs, 3)
	r := rand.New(rand.NewSource(1))
	for range 20 {
		mixed := slices.Clone(pkgs)
		r.Shuffle(len(mixed), func(i, j int) { mixed[i], mixed[j] = mixed[j], mixed[i] })
		mixed = append(mixed, mixed[0])
		if got := assign(mixed, 3); !slices.EqualFunc(got, want, slices.Equal[[]string]) {
			t.Fatalf("a reordered reading of the module changed the assignment:\n got %q\nwant %q", got, want)
		}
	}
}

// TestLargePackagesSpreadAcrossShards: the packages that decide how long a shard
// runs sit apart, and the measured load comes out level, the first shard's head
// start counted. Three shards are what ci.yml runs; the bound is a tenth over
// the mean, so a table edit that piles packages up fails here rather than on a
// slow runner.
func TestLargePackagesSpreadAcrossShards(t *testing.T) {
	pkgs := modulePackages(t)
	shards := assign(pkgs, 3)
	shardOf := func(rel string) int {
		for i, s := range shards {
			if slices.Contains(s, modulePath+rel) {
				return i
			}
		}
		t.Fatalf("%s is in no shard", rel)
		return -1
	}
	// The three heaviest go to three different runners.
	a, b, c := shardOf("internal/core/implement/loop"), shardOf("internal/core/ahoy"), shardOf("internal/surface/cli")
	if a == b || b == c || a == c {
		t.Errorf("implement/loop, ahoy and surface/cli are in shards %d, %d, %d; want three different ones", a, b, c)
	}
	loads := make([]int, 3)
	loads[0] = firstShardHead
	total := firstShardHead
	for i, s := range shards {
		for _, p := range s {
			loads[i] += weightOf(p)
			total += weightOf(p)
		}
	}
	if m := slices.Max(loads); m*30 > total*11 {
		t.Errorf("the shards' loads are %v; the heaviest is more than a tenth over the mean of %d", loads, total/3)
	}
}

// TestEveryShardStartsWithItsHeaviestPackage: go test starts packages in the
// order its arguments name them, so a shard names its slowest first and they
// start with the step rather than behind every faster package
// (iss-2609281715083637).
func TestEveryShardStartsWithItsHeaviestPackage(t *testing.T) {
	for i, shard := range assign(modulePackages(t), 3) {
		for j := 1; j < len(shard); j++ {
			if weightOf(shard[j-1]) < weightOf(shard[j]) {
				t.Errorf("shard %d names %s before the heavier %s", i+1, shard[j-1], shard[j])
			}
		}
	}
}

// TestEveryWeightedPackageIsInTheModule: a renamed or removed package would
// leave its measured weight behind and fall back to the default under its new
// name, unbalancing the shards with nothing said.
func TestEveryWeightedPackageIsInTheModule(t *testing.T) {
	pkgs := modulePackages(t)
	if len(weights) == 0 {
		t.Fatal("the weight table is empty")
	}
	for rel := range weights {
		if !slices.Contains(pkgs, modulePath+rel) {
			t.Errorf("the weight table names %s, which go list does not", rel)
		}
	}
}

// TestRunPrintsTheNamedShard: the shards the command prints, one call each, are
// assign's, one import path per line, and the union is the input.
func TestRunPrintsTheNamedShard(t *testing.T) {
	pkgs := modulePackages(t)
	list := func() ([]string, error) { return pkgs, nil }
	var got [][]string
	for shard := 1; shard <= 3; shard++ {
		var out, errw bytes.Buffer
		if code := run([]string{"-shard", fmt.Sprint(shard), "-of", "3"}, list, &out, &errw); code != 0 {
			t.Fatalf("shard %d exited %d: %s", shard, code, errw.String())
		}
		if !strings.Contains(errw.String(), fmt.Sprintf("shard %d of 3", shard)) {
			t.Errorf("shard %d's standard error does not name it: %q", shard, errw.String())
		}
		got = append(got, strings.Fields(out.String()))
	}
	checkPartition(t, pkgs, got)
	if want := assign(pkgs, 3); !slices.EqualFunc(got, want, slices.Equal[[]string]) {
		t.Errorf("the command printed %q; assign gives %q", got, want)
	}
}

// TestRunRefusesLoudly: everything that could leave a shard with nothing to
// test, or a package with no shard, exits 2 and prints nothing a step could
// pass to go test.
func TestRunRefusesLoudly(t *testing.T) {
	some := func() ([]string, error) { return []string{modulePath + "a", modulePath + "b"}, nil }
	for _, c := range []struct {
		name string
		args []string
		list func() ([]string, error)
		say  string
	}{
		{"no flags", nil, some, "-shard"},
		{"no count", []string{"-shard", "1"}, some, "-of"},
		{"no shard", []string{"-of", "3"}, some, "-shard"},
		{"shard zero", []string{"-shard", "0", "-of", "2"}, some, "from 1 to 2"},
		{"shard past the count", []string{"-shard", "3", "-of", "2"}, some, "from 1 to 2"},
		{"not a number", []string{"-shard", "one", "-of", "2"}, some, "shard"},
		{"an empty value, a matrix context that read as nothing", []string{"-shard", "", "-of", ""}, some, "shard"},
		{"a stray argument", []string{"-shard", "1", "-of", "2", "./..."}, some, "./..."},
		{"more shards than packages", []string{"-shard", "1", "-of", "3"}, some, "2 packages"},
		{"go list failed", []string{"-shard", "1", "-of", "2"}, func() ([]string, error) { return nil, errors.New("boom") }, "boom"},
		{"go list named nothing", []string{"-shard", "1", "-of", "2"}, func() ([]string, error) { return nil, nil }, "no packages"},
	} {
		t.Run(c.name, func(t *testing.T) {
			var out, errw bytes.Buffer
			if code := run(c.args, c.list, &out, &errw); code != 2 {
				t.Errorf("exit %d; want 2", code)
			}
			if out.Len() != 0 {
				t.Errorf("printed %q on standard output; a refusal prints nothing a step could test", out.String())
			}
			if !strings.Contains(errw.String(), c.say) {
				t.Errorf("standard error %q does not mention %q", errw.String(), c.say)
			}
		})
	}
}
