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

// Command dashboard-codegen-spike generates Plugin Framework Go (schema, model and conversion) for
// one dashboard panel type from Kibana's annotated OpenAPI document. It is the [tf-export 13] spike;
// see REPORT.md. Nothing in the provider uses it.
//
//	go run ./scripts/dashboard-codegen-spike extract  --oas ../kibana/oas_docs/output/kibana.yaml --panel-type slo_burn_rate --out testdata/slo_burn_rate.oas.json
//	go run ./scripts/dashboard-codegen-spike generate --oas testdata/slo_burn_rate.oas.json --panel-type slo_burn_rate \
//		--strategy computed --package sloburnratecomputed --out sloburnratecomputed/slo_burn_rate_gen.go
//	go run ./scripts/dashboard-codegen-spike survey   --oas ../kibana/oas_docs/output/kibana.yaml
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: dashboard-codegen-spike extract|generate|survey [flags]")
	}
	switch args[0] {
	case "extract":
		return cmdExtract(args[1:])
	case "generate":
		return cmdGenerate(args[1:])
	case "survey":
		return cmdSurvey(args[1:], stdout)
	}
	return fmt.Errorf("unknown command %q", args[0])
}

func cmdExtract(args []string) error {
	fs := flag.NewFlagSet("extract", flag.ContinueOnError)
	oasPath := fs.String("oas", "", "Kibana OpenAPI YAML")
	panelType := fs.String("panel-type", "", "panel discriminator, for example slo_burn_rate")
	out := fs.String("out", "", "output JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	s, err := loadSchemas(*oasPath)
	if err != nil {
		return err
	}
	subset := s.closure(panelTypeSchemaName(*panelType))
	b, err := json.MarshalIndent(map[string]any{"components": map[string]any{"schemas": subset}}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(*out, append(b, '\n'), 0o600)
}

func cmdGenerate(args []string) error {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	oasPath := fs.String("oas", "", "Kibana OpenAPI YAML, or JSON written by extract")
	panelType := fs.String("panel-type", "", "panel discriminator, for example slo_burn_rate")
	strategy := fs.String("strategy", strategyComputed, "how fields with a server default are generated: computed or preserve")
	pkg := fs.String("package", "", "Go package name")
	out := fs.String("out", "", "output Go file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	src, err := generateFile(*oasPath, *panelType, *strategy, *pkg)
	if err != nil {
		return err
	}
	return os.WriteFile(*out, src, 0o600)
}

// generateFile is generate without the file write, for the golden test.
func generateFile(oasPath, panelType, strategy, pkg string) ([]byte, error) {
	if strategy != strategyComputed && strategy != strategyPreserve {
		return nil, fmt.Errorf("unknown strategy %q", strategy)
	}
	s, err := loadSchemas(oasPath)
	if err != nil {
		return nil, err
	}
	cfg, component, err := s.configSchema(panelType)
	if err != nil {
		return nil, err
	}
	b := &builder{s: s}
	attrs := b.objectAttributes("config", cfg)
	if len(b.errs) > 0 {
		return nil, fmt.Errorf("unsupported:\n  %s", strings.Join(b.errs, "\n  "))
	}
	return generate(genOptions{
		Package:    pkg,
		PanelType:  panelType,
		BlockName:  panelType + "_config", // the panels union's key_template
		Schema:     component,
		KbapiType:  kbapiTypeName(component),
		Strategy:   strategy,
		BlockDescr: str(cfg["description"]),
	}, attrs)
}

// cmdSurvey answers, over the whole document, the questions the report needs evidence for:
// how `default` relates to `required`, how often a default is only stated in prose, and which panel
// types this generator could cover today.
func cmdSurvey(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("survey", flag.ContinueOnError)
	oasPath := fs.String("oas", "", "Kibana OpenAPI YAML")
	if err := fs.Parse(args); err != nil {
		return err
	}
	s, err := loadSchemas(*oasPath)
	if err != nil {
		return err
	}
	var panelTypes, roots []string
	for _, name := range s.names() {
		if pt, ok := strings.CutPrefix(name, panelTypeSchemaName("")); ok {
			panelTypes = append(panelTypes, pt)
		}
		if strings.Contains(strings.ToLower(name), "dashboard") {
			roots = append(roots, name)
		}
	}
	reach := s.closure(roots...)

	var withDefault, requiredAndDefault, proseOnly []string
	proseDefault := regexp.MustCompile(`(?i)\bdefaults? (to|is)\b`)
	var walk func(path string, n any)
	walk = func(path string, n any) {
		switch v := n.(type) {
		case map[string]any:
			if props, ok := v["properties"].(map[string]any); ok {
				req := map[string]bool{}
				if rs, ok := v["required"].([]any); ok {
					for _, r := range rs {
						req[str(r)] = true
					}
				}
				for name, p := range props {
					pm, _ := p.(map[string]any)
					if pm == nil {
						continue
					}
					switch _, has := pm["default"]; {
					case has && req[name]:
						requiredAndDefault = append(requiredAndDefault, path+"."+name)
						withDefault = append(withDefault, path+"."+name)
					case has:
						withDefault = append(withDefault, path+"."+name)
					case !req[name] && proseDefault.MatchString(str(pm["description"])):
						proseOnly = append(proseOnly, path+"."+name)
					}
				}
			}
			for k, c := range v {
				next := path
				if k != "properties" {
					next = path + "/" + k
				}
				walk(next, c)
			}
		case []any:
			for _, c := range v {
				walk(path, c)
			}
		}
	}
	for _, name := range reach.names() {
		walk(name, reach[name])
	}
	sort.Strings(proseOnly)
	sort.Strings(requiredAndDefault)

	fmt.Fprintf(stdout, "Component schemas reachable from the dashboard schemas: %d\n", len(reach))
	fmt.Fprintf(stdout, "Properties with a `default`: %d\n", len(withDefault))
	fmt.Fprintf(stdout, "Properties both in `required` and with a `default`: %d\n", len(requiredAndDefault))
	for _, p := range requiredAndDefault {
		fmt.Fprintf(stdout, "  %s\n", p)
	}
	fmt.Fprintf(stdout, "Optional properties whose description states a default but have no `default`: %d\n", len(proseOnly))
	for _, p := range proseOnly {
		fmt.Fprintf(stdout, "  %s\n", p)
	}

	fmt.Fprintf(stdout, "\nPanel types: %d\n", len(panelTypes))
	constructs := map[string]int{}
	for _, pt := range panelTypes {
		cfg, _, err := s.configSchema(pt)
		if err != nil {
			fmt.Fprintf(stdout, "  %-26s no config: %v\n", pt, err)
			continue
		}
		b := &builder{s: s}
		b.objectAttributes("config", cfg)
		if len(b.errs) == 0 {
			fmt.Fprintf(stdout, "  %-26s generates\n", pt)
			continue
		}
		kinds := map[string]int{}
		for _, e := range b.errs {
			k := e[strings.LastIndex(e, ": ")+2:]
			kinds[k]++
			constructs[k]++
		}
		fmt.Fprintf(stdout, "  %-26s %d unsupported: %s\n", pt, len(b.errs), summarize(kinds))
	}
	fmt.Fprintf(stdout, "\nUnsupported constructs across panel types: %s\n", summarize(constructs))
	return nil
}

func summarize(m map[string]int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if m[keys[i]] != m[keys[j]] {
			return m[keys[i]] > m[keys[j]]
		}
		return keys[i] < keys[j]
	})
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprintf("%s ×%d", k, m[k])
	}
	return strings.Join(parts, ", ")
}
