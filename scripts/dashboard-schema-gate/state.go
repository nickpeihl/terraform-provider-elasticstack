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

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// source records what the checked-in derived tree was derived from and which provider it was verified against.
type source struct {
	KibanaRepo string `json:"kibana_repo"`
	KibanaRef  string `json:"kibana_ref"`
	// ProviderVersion is the provider release the mapping was verified against: Kibana's flyout requires `>=` it.
	ProviderVersion string `json:"provider_version"`
	// ProviderCommit is the provider revision whose schema was compared.
	ProviderCommit string `json:"provider_commit"`
}

// knownMismatches maps each accepted kind mismatch line to the ticket that fixes it.
type knownMismatches map[string]string

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// writeJSON writes v indented and without HTML escaping, so `list<object>` stays readable in review.
func writeJSON(path string, v any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o600)
}

// readOptionalJSON reads v from path, reporting false when there is no such file.
func readOptionalJSON(path string, v any) (bool, error) {
	err := readJSON(path, v)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

type state struct {
	source   source
	known    knownMismatches
	baseline []string
}

func readState(dir string) (state, error) {
	var s state
	if err := readJSON(filepath.Join(dir, sourceFile), &s.source); err != nil {
		return s, err
	}
	if err := readJSON(filepath.Join(dir, knownMismatchesFile), &s.known); err != nil {
		return s, err
	}
	if err := readJSON(filepath.Join(dir, baselineFile), &s.baseline); err != nil {
		return s, err
	}
	return s, nil
}

func writeState(dir string, s state) error {
	sort.Strings(s.baseline)
	if s.known == nil {
		s.known = knownMismatches{}
	}
	if s.baseline == nil {
		s.baseline = []string{}
	}
	if err := writeJSON(filepath.Join(dir, sourceFile), s.source); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(dir, knownMismatchesFile), s.known); err != nil {
		return err
	}
	return writeJSON(filepath.Join(dir, baselineFile), s.baseline)
}
