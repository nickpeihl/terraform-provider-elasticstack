// Prototype (spike F): applies one or more OpenAPI Overlay 1.0 documents to an OAS in order, using
// speakeasy-api/openapi-overlay's ApplyToStrict so that any action whose JSONPath target matches
// nothing fails the run. The upstream CLI (`openapi-overlay apply`) uses the non-strict ApplyTo and
// silently ignores unmatched targets, which is exactly the drift signal we want.
//
//	go run . -spec kibana.yaml -out kibana.overlaid.yaml overlay1.yaml [overlay2.yaml ...]
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	"github.com/speakeasy-api/openapi-overlay/pkg/loader"
	"gopkg.in/yaml.v3"
)

func main() {
	spec := flag.String("spec", "", "input OAS (yaml/json)")
	out := flag.String("out", "", "output OAS (yaml)")
	flag.Parse()
	raw, err := os.ReadFile(*spec)
	must(err)
	var root yaml.Node
	must(yaml.Unmarshal(raw, &root))
	failed := false
	for _, path := range flag.Args() {
		o, err := loader.LoadOverlay(path)
		must(err)
		must(o.Validate())
		err, warnings := o.ApplyToStrict(&root)
		for _, w := range warnings {
			fmt.Fprintf(os.Stderr, "WARN %s: %s\n", path, w)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "UNMATCHED %s: %v\n", path, err)
			failed = true
		}
		fmt.Fprintf(os.Stderr, "applied %s (%d actions)\n", path, len(o.Actions))
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	must(enc.Encode(&root))
	must(os.WriteFile(*out, buf.Bytes(), 0o644))
	if failed {
		os.Exit(3)
	}
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
