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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGeneratedCodeIsCurrent regenerates both strategies from the checked-in OpenAPI extract and
// compares with the checked-in code.
func TestGeneratedCodeIsCurrent(t *testing.T) {
	for _, s := range []struct{ strategy, pkg string }{
		{strategyComputed, "sloburnratecomputed"},
		{strategyPreserve, "sloburnratepreserve"},
	} {
		got, err := generateFile(filepath.Join("testdata", "slo_burn_rate.oas.json"), "slo_burn_rate", s.strategy, s.pkg)
		require.NoError(t, err)
		path := filepath.Join(s.pkg, "slo_burn_rate_gen.go")
		if *update {
			require.NoError(t, os.WriteFile(path, got, 0o600))
			continue
		}
		want, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, string(want), string(got), "%s is stale; rerun with -update", path)
	}
}
