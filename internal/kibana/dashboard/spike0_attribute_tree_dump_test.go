// Spike 0 (dashboards-terraform): dumps the baseline elasticstack_kibana_dashboard schema as a
// normalized "attribute tree" JSON, filtered to the comparison slice.
// Scratch-branch only (spike0/attribute-tree-dump); never pushed.
//
// Run: SPIKE0_OUT=/abs/path/golden-attribute-tree.json go test ./internal/kibana/dashboard/ -run TestSpike0DumpAttributeTree -count=1

package dashboard

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"unsafe"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

type s0Attr struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	Element string `json:"element,omitempty"`
	Mode    string `json:"mode"`
}

type s0Discriminator struct {
	Path   string              `json:"path"`
	Values map[string][]string `json:"values"`
}

type s0Group struct {
	Kind          string           `json:"kind"`
	Members       []string         `json:"members"`
	Discriminator *s0Discriminator `json:"discriminator,omitempty"`
}

type s0Tree struct {
	Format     string    `json:"format"`
	Resource   string    `json:"resource"`
	Source     string    `json:"source"`
	Attributes []s0Attr  `json:"attributes"`
	Groups     []s0Group `json:"groups"`
}

type s0Walker struct {
	attrs  []s0Attr
	groups map[string]*s0Group
	// discriminator conditions collected from AllowedIf validators: dependent path -> member -> values
	disc   map[string]map[string][]string
	opaque map[string][]string
}

// Slice filter: these nodes are emitted (so union variant keys are visible) but their subtree is pruned.
func s0Prune(parent, name string) bool {
	isPanel := parent == "panels[]" || parent == "sections[].panels[]"
	if isPanel && (name == "config_json" || strings.HasSuffix(name, "_config")) {
		return name != "image_config" && name != "vis_config"
	}
	if strings.HasSuffix(parent, ".vis_config.by_value") {
		return name != "xy_chart_config" && name != "metric_chart_config"
	}
	if parent == "pinned_panels[]" && strings.HasSuffix(name, "_config") {
		return name != "range_slider_control_config"
	}
	return false
}

func s0Join(parent, name string) string {
	if parent == "" {
		return name
	}
	return parent + "." + name
}

// pathExpr builds a framework expression for our path notation ("a[].b").
func s0PathExpr(p string) path.Expression {
	e := path.MatchRoot(strings.Split(p, ".")[0])
	first := true
	for _, seg := range strings.Split(p, ".") {
		name := strings.TrimSuffix(seg, "[]")
		if first {
			e = path.MatchRoot(name)
			first = false
		} else {
			e = e.AtName(name)
		}
		if strings.HasSuffix(seg, "[]") {
			e = e.AtAnyListIndex()
		}
	}
	return e
}

func s0ExprToPath(e path.Expression) string {
	var b strings.Builder
	for _, st := range e.Resolve().Steps() {
		switch s := st.(type) {
		case path.ExpressionStepAttributeNameExact:
			if b.Len() > 0 {
				b.WriteString(".")
			}
			b.WriteString(string(s))
		case path.ExpressionStepElementKeyIntAny, path.ExpressionStepElementKeyValueAny:
			b.WriteString("[]")
		default:
			b.WriteString(fmt.Sprintf("<%T>", st))
		}
	}
	return b.String()
}

// s0Field reads a (possibly unexported) struct field from an arbitrary value.
func s0Field(v any, name string) (reflect.Value, bool) {
	rv := reflect.ValueOf(v)
	cp := reflect.New(rv.Type()).Elem()
	cp.Set(rv)
	if cp.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}
	f := cp.FieldByName(name)
	if !f.IsValid() {
		return reflect.Value{}, false
	}
	return reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem(), true
}

func (w *s0Walker) addGroup(kind string, members []string) {
	ms := append([]string(nil), members...)
	sort.Strings(ms)
	key := kind + "|" + strings.Join(ms, ",")
	if _, ok := w.groups[key]; !ok {
		w.groups[key] = &s0Group{Kind: kind, Members: ms}
	}
}

func (w *s0Walker) validators(p string, attr any) {
	rv := reflect.ValueOf(attr)
	vf := rv.FieldByName("Validators")
	if !vf.IsValid() {
		return
	}
	for i := 0; i < vf.Len(); i++ {
		v := vf.Index(i).Interface()
		t := reflect.TypeOf(v)
		switch t.Name() {
		case "ConflictsWithValidator", "ExactlyOneOfValidator", "AtLeastOneOfValidator":
			exprs, _ := s0Field(v, "PathExpressions")
			members := []string{p}
			for j := 0; j < exprs.Len(); j++ {
				e := exprs.Index(j).Interface().(path.Expression)
				members = append(members, s0ExprToPath(s0PathExpr(p).Merge(e)))
			}
			kind := map[string]string{"ConflictsWithValidator": "conflicts_with", "ExactlyOneOfValidator": "exactly_one_of", "AtLeastOneOfValidator": "at_least_one_of"}[t.Name()]
			w.addGroup(kind, members)
		case "exactlyOneOfNestedAttrsValidator":
			opts, _ := s0Field(v, "opts")
			names := opts.FieldByName("AttrNames")
			var members []string
			for j := 0; j < names.Len(); j++ {
				members = append(members, s0Join(p, names.Index(j).String()))
			}
			w.addGroup("exactly_one_of", members)
		case "Condition":
			exprPtr, ok := s0Field(v, "dependentPathExpression")
			if !ok || exprPtr.IsNil() {
				w.opaque[p] = append(w.opaque[p], "Condition(static path)")
				continue
			}
			dep := s0ExprToPath(s0PathExpr(p).Merge(*(exprPtr.Interface().(*path.Expression))))
			if presence, _ := s0Field(v, "matchNestedObjectPresence"); presence.Bool() {
				// ForbiddenIfDrilldownVariantSiblingNestedPresent: pairwise sibling exclusion.
				w.addGroup("conflicts_with", []string{p, dep})
				continue
			}
			allowed, _ := s0Field(v, "allowedIfOptions")
			if allowed.IsNil() {
				continue // RequiredIf: same values as AllowedIf for typed config blocks; not modelled separately
			}
			vals, _ := s0Field(v, "allowedValues")
			if w.disc[dep] == nil {
				w.disc[dep] = map[string][]string{}
			}
			for j := 0; j < vals.Len(); j++ {
				w.disc[dep][p] = append(w.disc[dep][p], vals.Index(j).String())
			}
		case "DrilldownItemModeValidator":
			// Opaque; semantics from its Description(): exactly one of dashboard/discover/url per item.
			w.addGroup("exactly_one_of", []string{s0Join(p, "dashboard"), s0Join(p, "discover"), s0Join(p, "url")})
		case "DrilldownListItemVariantsValidator":
			// Opaque; semantics from its Description(): at least one variant (exclusion comes from sibling Conditions).
			w.addGroup("at_least_one_of", []string{s0Join(p, "dashboard_drilldown"), s0Join(p, "discover_drilldown"), s0Join(p, "url_drilldown")})
		default:
			w.opaque[p] = append(w.opaque[p], t.String())
		}
	}
}

func s0Mode(attr schema.Attribute) string {
	switch {
	case attr.IsRequired():
		return "required"
	case attr.IsOptional() && attr.IsComputed():
		return "optional_computed"
	case attr.IsOptional():
		return "optional"
	default:
		return "computed"
	}
}

func s0ElemKind(t fmt.Stringer) string {
	s := t.String()
	switch {
	case strings.Contains(s, "String"):
		return "string"
	case strings.Contains(s, "Bool"):
		return "bool"
	case strings.Contains(s, "Int"), strings.Contains(s, "Float"), strings.Contains(s, "Number"):
		return "number"
	}
	return s
}

func (w *s0Walker) walk(parent string, attrs map[string]schema.Attribute) {
	for name, attr := range attrs {
		p := s0Join(parent, name)
		a := s0Attr{Path: p, Mode: s0Mode(attr)}
		var children map[string]schema.Attribute
		childPrefix := p
		switch at := attr.(type) {
		case schema.StringAttribute:
			a.Kind = "string"
			if at.CustomType != nil && strings.Contains(strings.ToLower(fmt.Sprintf("%T", at.CustomType)), "json") {
				a.Kind = "json-string"
			}
		case schema.BoolAttribute:
			a.Kind = "bool"
		case schema.Int64Attribute, schema.Int32Attribute, schema.Float64Attribute, schema.Float32Attribute, schema.NumberAttribute:
			a.Kind = "number"
		case schema.ListAttribute:
			a.Kind, a.Element = "list", s0ElemKind(at.ElementType)
		case schema.SetAttribute:
			a.Kind, a.Element = "set", s0ElemKind(at.ElementType)
		case schema.MapAttribute:
			a.Kind, a.Element = "map", s0ElemKind(at.ElementType)
		case schema.SingleNestedAttribute:
			a.Kind, children = "object", at.Attributes
		case schema.ListNestedAttribute:
			a.Kind, a.Element, children, childPrefix = "list", "object", at.NestedObject.Attributes, p+"[]"
			w.validatorsNested(childPrefix, at.NestedObject)
		case schema.SetNestedAttribute:
			a.Kind, a.Element, children, childPrefix = "set", "object", at.NestedObject.Attributes, p+"[]"
			w.validatorsNested(childPrefix, at.NestedObject)
		case schema.MapNestedAttribute:
			a.Kind, a.Element, children, childPrefix = "map", "object", at.NestedObject.Attributes, p+"{}"
			w.validatorsNested(childPrefix, at.NestedObject)
		default:
			a.Kind = fmt.Sprintf("UNKNOWN(%T)", attr)
		}
		w.attrs = append(w.attrs, a)
		pruned := s0Prune(parent, name)
		// Pruned variant keys still contribute their sibling-level validators (ConflictsWith / AllowedIf).
		w.validators(p, attr)
		if children != nil && !pruned {
			w.walk(childPrefix, children)
		}
	}
}

func (w *s0Walker) validatorsNested(p string, obj schema.NestedAttributeObject) {
	w.validators(p, obj)
}

func TestSpike0DumpAttributeTree(t *testing.T) {
	out := os.Getenv("SPIKE0_OUT")
	if out == "" {
		t.Skip("SPIKE0_OUT not set")
	}
	s := getSchema()
	w := &s0Walker{groups: map[string]*s0Group{}, disc: map[string]map[string][]string{}, opaque: map[string][]string{}}
	w.walk("", s.Attributes) // Blocks (kibana_connection) intentionally excluded: provider plumbing, not dashboard API.

	// pinnedPanelControlValidator is an opaque object validator; derive its exactly-one-of group and
	// type -> key mapping from the same registry function the validator uses.
	pinnedMembers := []string{}
	pinnedDisc := map[string][]string{}
	for _, n := range pinnedPanelControlConfigNames {
		pinnedMembers = append(pinnedMembers, "pinned_panels[]."+n)
	}
	for _, h := range AllHandlers() {
		if key, ok := pinnedPanelExpectedTypedControlAttr(h.PanelType()); ok {
			pinnedDisc["pinned_panels[]."+key] = []string{h.PanelType()}
		}
	}
	w.addGroup("exactly_one_of", pinnedMembers)

	w.disc["pinned_panels[].type"] = pinnedDisc
	w.mergeConflictCliques()

	// Attach each discriminator to the group whose member set equals the discriminated keys.
	for dep, vals := range w.disc {
		attached := false
		for _, g := range w.groups {
			if len(g.Members) != len(vals) {
				continue
			}
			all := true
			for m := range vals {
				if !s0Contains(g.Members, m) {
					all = false
				}
			}
			if all {
				g.Discriminator = &s0Discriminator{Path: dep, Values: vals}
				attached = true
			}
		}
		if !attached {
			t.Errorf("discriminator %s matched no group", dep)
		}
	}
	// Normalize: a conflicts_with group made redundant by an exactly_one_of group over the same members is dropped.
	for k, g := range w.groups {
		if g.Kind == "conflicts_with" {
			if _, ok := w.groups["exactly_one_of|"+strings.Join(g.Members, ",")]; ok {
				delete(w.groups, k)
			}
		}
	}

	tree := s0Tree{
		Format:   "attribute-tree/v1",
		Resource: "elasticstack_kibana_dashboard",
		Source:   os.Getenv("SPIKE0_SOURCE"),
	}
	sort.Slice(w.attrs, func(i, j int) bool { return w.attrs[i].Path < w.attrs[j].Path })
	tree.Attributes = w.attrs
	inTree := map[string]bool{}
	for _, a := range w.attrs {
		inTree[a.Path] = true
	}
	for _, g := range w.groups {
		keep := true
		for _, m := range g.Members {
			if !inTree[m] {
				keep = false // group lives inside a pruned (out-of-slice) subtree
			}
		}
		if keep {
			tree.Groups = append(tree.Groups, *g)
		}
	}
	sort.Slice(tree.Groups, func(i, j int) bool {
		a, b := tree.Groups[i], tree.Groups[j]
		if a.Members[0] != b.Members[0] {
			return a.Members[0] < b.Members[0]
		}
		return a.Kind < b.Kind
	})
	b, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, append(b, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	ob, _ := json.MarshalIndent(w.opaque, "", "  ")
	if err := os.WriteFile(strings.TrimSuffix(out, ".json")+".opaque-validators.json", append(ob, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %d attributes, %d groups", len(tree.Attributes), len(tree.Groups))
}

func s0Contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

// mergeConflictCliques folds pairwise conflicts_with groups whose members form a clique into one group, and
// a conflicts_with clique + at_least_one_of over the same members into exactly_one_of.
func (w *s0Walker) mergeConflictCliques() {
	edges := map[string]map[string]bool{}
	pairKeys := map[string]bool{}
	for k, g := range w.groups {
		if g.Kind != "conflicts_with" || len(g.Members) != 2 {
			continue
		}
		a, b := g.Members[0], g.Members[1]
		if edges[a] == nil {
			edges[a] = map[string]bool{}
		}
		if edges[b] == nil {
			edges[b] = map[string]bool{}
		}
		edges[a][b], edges[b][a] = true, true
		pairKeys[k] = true
	}
	seen := map[string]bool{}
	for start := range edges {
		if seen[start] {
			continue
		}
		comp := []string{}
		stack := []string{start}
		for len(stack) > 0 {
			n := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if seen[n] {
				continue
			}
			seen[n] = true
			comp = append(comp, n)
			for m := range edges[n] {
				stack = append(stack, m)
			}
		}
		if len(comp) < 3 {
			continue
		}
		clique := true
		for _, a := range comp {
			for _, b := range comp {
				if a != b && !edges[a][b] {
					clique = false
				}
			}
		}
		if !clique {
			continue
		}
		sort.Strings(comp)
		for k, g := range w.groups {
			if pairKeys[k] && s0Contains(comp, g.Members[0]) {
				delete(w.groups, k)
			}
		}
		alo := "at_least_one_of|" + strings.Join(comp, ",")
		if _, ok := w.groups[alo]; ok {
			delete(w.groups, alo)
			w.addGroup("exactly_one_of", comp)
		} else {
			w.addGroup("conflicts_with", comp)
		}
	}
}
