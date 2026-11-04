// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// Command dashboard-schema-gate compares the elasticstack_kibana_dashboard schema that Kibana's variant mapping
// derives from its published OpenAPI document with the provider's real schema. It fails on kind mismatches (they
// break `terraform plan`) that are not on the known list, and on any other difference not in the baseline. Both lists
// may shrink and never grow. See README.md.
//
// Usage:
//
//	go run ./scripts/dashboard-schema-gate check [--dir DIR] [--base-ref REF | --base-dir DIR] [--kibana-dir KIBANA]
//	go run ./scripts/dashboard-schema-gate update --kibana-dir KIBANA [--kibana-repo OWNER/REPO] [--kibana-ref REF] [--tree FILE] [--provider-version V] [--provider-commit SHA]
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Files of the gate's checked-in state, relative to --dir.
const (
	sourceFile          = "source.json"
	derivedTreeFile     = "derived_attribute_tree.json"
	knownMismatchesFile = "known_kind_mismatches.json"
	baselineFile        = "baseline.json"
)

// defaultDir is where the state lives, relative to the repository root `go run` is started from.
const defaultDir = "scripts/dashboard-schema-gate"

// untracked is the ticket `update` records for a kind mismatch nobody has filed yet.
const untracked = "untracked: file a ticket"

var errCheckFailed = errors.New("the dashboard schema gate failed")

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) > 0 {
		switch args[0] {
		case "check":
			return cmdCheck(args[1:], stdout, stderr)
		case "update":
			return cmdUpdate(args[1:], stdout, stderr)
		}
	}
	return errors.New("usage: dashboard-schema-gate check|update [flags]")
}

func cmdCheck(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", defaultDir, "directory holding the gate's state")
	baseRef := fs.String("base-ref", "", "git ref whose lists the current lists may not grow beyond (for example origin/main)")
	baseDir := fs.String("base-dir", "", "directory holding the lists the current lists may not grow beyond")
	kibanaDir := fs.String("kibana-dir", "", "derive from this Kibana checkout instead of the checked-in tree (needs node)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	st, err := readState(*dir)
	if err != nil {
		return err
	}
	treePath := filepath.Join(*dir, derivedTreeFile)
	if *kibanaDir != "" {
		if treePath, err = deriveToTemp(*kibanaDir); err != nil {
			return err
		}
		defer os.Remove(treePath)
	}
	diffs, err := diffsFor(treePath)
	if err != nil {
		return err
	}

	var problems []string
	add := func(format string, lines []string) {
		for _, l := range lines {
			problems = append(problems, fmt.Sprintf(format, l))
		}
	}
	knownLines := keys(st.known)
	add("new kind mismatch (breaks terraform plan): %s\n    Fix the mapping in Kibana or the provider's schema; the known list may not grow.", missingFrom(diffs.kind, knownLines))
	add("fixed kind mismatch: %s\n    Remove it from "+knownMismatchesFile+" (run update).", missingFrom(knownLines, diffs.kind))
	add("new difference: %s\n    The baseline may not grow; fix the mapping in Kibana or the provider's schema.", missingFrom(diffs.other, st.baseline))
	// A difference under a new kind mismatch is hidden by it, not fixed.
	add("fixed difference: %s\n    Remove it from "+baselineFile+" (run update).", notUnder(missingFrom(st.baseline, diffs.other), diffs.kind))

	base, haveBase, err := readBase(*dir, *baseRef, *baseDir)
	if err != nil {
		return err
	}
	if haveBase {
		add(knownMismatchesFile+" grew: %s\n    The known kind mismatches may shrink and never grow.", missingFrom(knownLines, keys(base.known)))
		// A kind mismatch fixed since the base hid the differences at and under it; they may enter the baseline.
		fixed := missingFrom(keys(base.known), knownLines)
		add(baselineFile+" grew: %s\n    The baseline may shrink and never grow.", notUnder(missingFrom(st.baseline, base.baseline), fixed))
	}

	for _, p := range problems {
		fmt.Fprintln(stderr, p)
	}
	if len(problems) > 0 {
		return fmt.Errorf("%w with %d problem(s)", errCheckFailed, len(problems))
	}
	fmt.Fprintf(stdout, "dashboard schema gate passed: %d known kind mismatch(es), %d baseline difference(s); verified against provider %s\n",
		len(knownLines), len(st.baseline), st.source.ProviderVersion)
	return nil
}

func cmdUpdate(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", defaultDir, "directory holding the gate's state")
	kibanaDir := fs.String("kibana-dir", "", "Kibana checkout to derive from; its provider-version constant is updated (required)")
	kibanaRepo := fs.String("kibana-repo", "", "repository --kibana-ref is in, recorded in "+sourceFile+" (default: the recorded one)")
	kibanaRef := fs.String("kibana-ref", "", "Kibana revision recorded in "+sourceFile+" (default: HEAD of --kibana-dir)")
	treePath := fs.String("tree", "", "use this attribute tree instead of deriving it from --kibana-dir")
	providerVersion := fs.String("provider-version", "", "provider release verified (default: VERSION in the Makefile)")
	providerCommit := fs.String("provider-commit", "", "provider revision compared (default: HEAD)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *kibanaDir == "" {
		return errors.New("update: --kibana-dir is required: the flyout's provider version lives in Kibana and is updated with the check")
	}

	var err error
	if *kibanaRef == "" {
		if *kibanaRef, err = gitOutput(*kibanaDir, "rev-parse", "HEAD"); err != nil {
			return err
		}
	}
	if *providerVersion == "" {
		if *providerVersion, err = makefileVersion("Makefile"); err != nil {
			return err
		}
	}
	if *providerCommit == "" {
		if *providerCommit, err = gitOutput(".", "rev-parse", "--short=12", "HEAD"); err != nil {
			return err
		}
	}

	var treeJSON []byte
	if *treePath != "" {
		treeJSON, err = os.ReadFile(*treePath)
	} else {
		treeJSON, err = deriveTree(kibanaSource{dir: *kibanaDir})
	}
	if err != nil {
		return err
	}
	treeOut := filepath.Join(*dir, derivedTreeFile)
	if err := os.WriteFile(treeOut, treeJSON, 0o600); err != nil {
		return err
	}
	diffs, err := diffsFor(treeOut)
	if err != nil {
		return err
	}

	var old state
	_, _ = readOptionalJSON(filepath.Join(*dir, sourceFile), &old.source)
	if *kibanaRepo == "" {
		*kibanaRepo = old.source.KibanaRepo
	}
	_, _ = readOptionalJSON(filepath.Join(*dir, knownMismatchesFile), &old.known)
	_, _ = readOptionalJSON(filepath.Join(*dir, baselineFile), &old.baseline)
	next := state{
		source:   source{KibanaRepo: *kibanaRepo, KibanaRef: *kibanaRef, ProviderVersion: *providerVersion, ProviderCommit: *providerCommit},
		known:    knownMismatches{},
		baseline: diffs.other,
	}
	for _, l := range diffs.kind {
		ticket, ok := old.known[l]
		if !ok {
			ticket = untracked
		}
		next.known[l] = ticket
	}
	if err := writeState(*dir, next); err != nil {
		return err
	}
	if err := writeKibanaProviderVersion(filepath.Join(*kibanaDir, kibanaProviderTS), *providerVersion, *providerCommit); err != nil {
		return err
	}

	report := func(what string, lines []string) {
		for _, l := range lines {
			fmt.Fprintf(stdout, "%s: %s\n", what, l)
		}
	}
	report("added kind mismatch", missingFrom(diffs.kind, keys(old.known)))
	report("removed kind mismatch", missingFrom(keys(old.known), diffs.kind))
	report("added difference", missingFrom(diffs.other, old.baseline))
	report("removed difference", missingFrom(old.baseline, diffs.other))
	fmt.Fprintf(stdout, "updated: %d kind mismatch(es), %d difference(s); Kibana's TERRAFORM_PROVIDER is now %s\n",
		len(diffs.kind), len(diffs.other), *providerVersion)
	return nil
}

// deriveToTemp writes the tree derived from a Kibana checkout to a temporary file.
func deriveToTemp(kibanaDir string) (string, error) {
	b, err := deriveTree(kibanaSource{dir: kibanaDir})
	if err != nil {
		return "", err
	}
	f, err := os.CreateTemp("", "derived-attribute-tree-*.json")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.Write(b); err != nil {
		return "", err
	}
	return f.Name(), nil
}

func diffsFor(treePath string) (differences, error) {
	derived, err := readTree(treePath)
	if err != nil {
		return differences{}, err
	}
	provider, err := providerTree(context.Background())
	if err != nil {
		return differences{}, err
	}
	return compareTrees(derived, provider), nil
}

// readBase reads the lists the current ones are compared against: from a directory, or from a git ref. A list that
// does not exist there yet sets no limit.
func readBase(dir, ref, baseDir string) (state, bool, error) {
	var base state
	switch {
	case baseDir != "":
		okKnown, err := readOptionalJSON(filepath.Join(baseDir, knownMismatchesFile), &base.known)
		if err != nil {
			return base, false, err
		}
		okBaseline, err := readOptionalJSON(filepath.Join(baseDir, baselineFile), &base.baseline)
		if err != nil {
			return base, false, err
		}
		return base, okKnown && okBaseline, nil
	case ref != "":
		if _, err := gitOutput(".", "rev-parse", "--verify", "--quiet", ref+"^{commit}"); err != nil {
			return base, false, fmt.Errorf("base ref %q does not resolve (is the base branch fetched?): %w", ref, err)
		}
		tmp, err := os.MkdirTemp("", "dashboard-schema-gate-base-")
		if err != nil {
			return base, false, err
		}
		defer os.RemoveAll(tmp)
		for _, name := range []string{knownMismatchesFile, baselineFile} {
			// The ref resolves, so a failure here means the list is not on the base yet: it sets no limit.
			out, err := exec.Command("git", "show", ref+":"+filepath.ToSlash(filepath.Join(dir, name))).Output()
			if err != nil {
				return base, false, nil //nolint:nilerr // see above
			}
			if err := os.WriteFile(filepath.Join(tmp, name), out, 0o600); err != nil {
				return base, false, err
			}
		}
		return readBase(dir, "", tmp)
	}
	return base, false, nil
}

var (
	providerVersionRe = regexp.MustCompile(`(TERRAFORM_PROVIDER\s*=\s*\{[^}]*version:\s*')[^']*(')`)
	verifiedCommentRe = regexp.MustCompile(`// \S+ was the latest release when the mapping was compared with provider main at [0-9a-f]+;`)
	makefileVersionRe = regexp.MustCompile(`(?m)^VERSION \?= (\S+)$`)
)

// writeKibanaProviderVersion sets the provider version Kibana's flyout requires (`>=`), in Kibana's provider.ts.
func writeKibanaProviderVersion(path, version, commit string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("kibana provider version: %w", err)
	}
	if !providerVersionRe.Match(b) {
		return fmt.Errorf("%s: no TERRAFORM_PROVIDER version to update", path)
	}
	b = providerVersionRe.ReplaceAll(b, []byte("${1}"+version+"${2}"))
	b = verifiedCommentRe.ReplaceAllLiteral(b, []byte(fmt.Sprintf(
		"// %s was the latest release when the mapping was compared with provider main at %s;", version, commit)))
	return os.WriteFile(path, b, 0o600)
}

func makefileVersion(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	m := makefileVersionRe.FindSubmatch(b)
	if m == nil {
		return "", fmt.Errorf("%s: no VERSION", path)
	}
	return string(m[1]), nil
}

func gitOutput(dir string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	if err != nil {
		return "", fmt.Errorf("git %s in %s: %w", strings.Join(args, " "), dir, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// linePath is the attribute path a difference line is about ("mode a.b: …" -> "a.b").
func linePath(line string) string {
	_, rest, _ := strings.Cut(line, " ")
	path, _, _ := strings.Cut(rest, ":")
	return path
}

// notUnder drops the lines about one of the attributes the parent lines are about, or about attributes inside one.
func notUnder(lines, parents []string) []string {
	var out []string
	for _, l := range lines {
		p := linePath(l)
		hidden := false
		for _, parent := range parents {
			pp := linePath(parent)
			if p == pp || strings.HasPrefix(p, pp+".") || strings.HasPrefix(p, pp+"[].") || strings.HasPrefix(p, pp+"{}.") {
				hidden = true
			}
			// A mismatch under panels[] hides the same attribute inside sections too.
			if strings.HasPrefix(pp, panels+".") && (p == sectionPanels+strings.TrimPrefix(pp, panels) || strings.HasPrefix(p, sectionPanels+strings.TrimPrefix(pp, panels)+".")) {
				hidden = true
			}
		}
		if !hidden {
			out = append(out, l)
		}
	}
	return out
}

// missingFrom lists the lines of a that are not in b.
func missingFrom(a, b []string) []string {
	in := map[string]bool{}
	for _, l := range b {
		in[l] = true
	}
	var out []string
	for _, l := range a {
		if !in[l] {
			out = append(out, l)
		}
	}
	return out
}

func keys(m knownMismatches) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
