// PROTOTYPE (spike F): schema-conversion consumer. Reads the overlaid OAS and emits an attribute tree
// (spike0 format attribute-tree/v1) for the dashboard resource.
//   node src/emit_attribute_tree.ts <overlaid.yaml> <out.json> [--full]
// Without --full it applies spike0's slice filter (only image/vis panels, xy/metric charts and the
// range slider control are expanded); with --full it walks everything (used by the lint).
import { writeFileSync } from 'node:fs';
import {
  type Kind,
  type Mode,
  type Schema,
  type UnionX,
  branchesOf,
  defaultMode,
  fieldSpecs,
  isUnion,
  lint,
  loadDoc,
  mergeBranches,
  norm,
  primKind,
  schemas,
  splitBranches,
  variantsOf,
  withPolicy,
  x,
} from './f_core.ts';

const [, , input, output, ...flags] = process.argv;
const FULL = flags.includes('--full');

interface Attr { path: string; kind: Kind; element?: Kind; mode: Mode }
interface Group { kind: string; members: string[]; discriminator?: { path: string; values: Record<string, string[]> } }
const attrs: Attr[] = [];
const groups: Group[] = [];
const lock: Record<string, unknown> = {};

// spike0 slice filter (measuring-stick only, not part of the mapping)
const SLICE_EXPAND: Record<string, string[]> = {
  'panels[]': ['image_config', 'vis_config'],
  'sections[].panels[]': ['image_config', 'vis_config'],
  'panels[].vis_config.by_value': ['xy_chart_config', 'metric_chart_config'],
  'sections[].panels[].vis_config.by_value': ['xy_chart_config', 'metric_chart_config'],
  'pinned_panels[]': ['range_slider_control_config'],
};
const expandVariant = (parent: string, key: string): boolean =>
  FULL || !SLICE_EXPAND[parent] || SLICE_EXPAND[parent].includes(key);

const join = (p: string, n: string): string => (p ? `${p}.${n}` : n);

/** Emits the attributes of an object-like TF node at `path` (children only). Returns hoisted fields' emitters. */
const emitObjectBody = (s: Schema, path: string, recognizerToDrop?: string, prefix = ''): Array<() => void> =>
  withPolicy(s, () => emitObjectBodyInner(s, path, recognizerToDrop, prefix));
const emitObjectBodyInner = (s: Schema, path: string, recognizerToDrop?: string, prefix = ''): Array<() => void> => {
  if (isUnion(s)) return emitUnionBody(s, path, recognizerToDrop);
  const hoisted: Array<() => void> = [];
  for (const f of fieldSpecs(s, recognizerToDrop)) {
    const name = prefix + f.tfName;
    if (f.op === 'flatten') {
      hoisted.push(...emitObjectBody(f.schema, path, undefined, prefix + (f.prefix ?? '')));
      continue;
    }
    const target = f.hoist ? () => path.split('.').slice(0, -1).join('.') : () => path;
    const run = (): void => {
      const p = join(target(), name);
      const mode = defaultMode(f);
      if (f.op === 'json') attrs.push({ path: p, kind: 'json-string', mode });
      else if (f.op === 'unwrap') attrs.push({ path: p, kind: primKind(norm(f.schema.properties[f.unwrap!])), mode });
      else emitValue(f.schema, p, mode);
    };
    if (f.hoist) hoisted.push(run);
    else run();
  }
  // hoisted fields from flattened children land here unless they are themselves hoisting further
  return hoisted;
};

/** Emits one attribute at `p` for schema `s` (and its subtree). */
const emitValue = (s: Schema, p: string, mode: Mode): void => withPolicy(s, () => emitValueInner(s, p, mode));
const emitValueInner = (s: Schema, p: string, mode: Mode): void => {
  const custom = x(s, 'custom');
  if (custom) {
    attrs.push({ path: p, kind: custom.kind, element: 'object', mode: custom.mode ?? mode });
    for (const a of custom.attributes) attrs.push({ path: `${p}[].${a.name}`, kind: a.kind, mode: a.mode });
    return;
  }
  if (x(s, 'json')) return void attrs.push({ path: p, kind: 'json-string', mode });
  if (s.type === 'array') {
    const items = norm(s.items);
    const wrap = x(items, 'json')?.wrap;
    if (wrap) {
      attrs.push({ path: p, kind: 'list', element: 'object', mode });
      attrs.push({ path: `${p}[].${wrap}`, kind: 'json-string', mode: 'required' });
      return;
    }
    if (isUnion(items) || items.properties) {
      attrs.push({ path: p, kind: 'list', element: 'object', mode });
      runAll(emitObjectBody(items, `${p}[]`));
      return;
    }
    attrs.push({ path: p, kind: 'list', element: primKind(items), mode });
    return;
  }
  if (isUnion(s) || s.properties) {
    attrs.push({ path: p, kind: 'object', mode });
    runAll(emitObjectBody(s, p));
    return;
  }
  attrs.push({ path: p, kind: primKind(s), mode });
};
const runAll = (fs: Array<() => void>): void => fs.forEach((f) => f());

const emitUnionBody = (s: Schema, path: string, recognizerToDrop?: string): Array<() => void> => {
  const u: UnionX | undefined = x(s, 'union');
  if (!u) {
    lint.unmapped.push(`${path}: union (${branchesOf(s).map((b) => b['x-tf-ref'] ?? b.title ?? 'inline').join(' | ')}) has no x-tf-union / x-tf-json`);
    return [];
  }
  if (u.mode === 'merge') return emitObjectBody(mergeBranches(branchesOf(s)), path, recognizerToDrop);
  const variants = variantsOf(s, u, path);
  const r = u.recognizer!;
  // siblings: properties common to the branches, other than the content property (only with `content`)
  if (u.content || u.keep === 'sibling') {
    const first = variants[0].branches[0];
    const sib: Schema = { properties: {}, required: first.required ?? [] };
    for (const [k, v] of Object.entries(first.properties ?? {})) {
      if (k === u.content) continue;
      if (k === r.property && u.keep !== 'sibling') continue;
      if (!u.content && k !== r.property) continue;
      sib.properties[k] = k === r.property ? { type: 'string' } : v;
    }
    runAll(emitObjectBody(sib, path));
  }
  const members: string[] = [];
  const values: Record<string, string[]> = {};
  lock[path] = Object.fromEntries(
    variants.map((v) => [v.key, { values: v.values, branches: v.branches.map((b) => b['x-tf-ref'] ?? b.title ?? 'inline') }]),
  );
  for (const v of variants) {
    const p = join(path, v.key);
    members.push(p);
    values[p] = v.values;
    attrs.push({ path: p, kind: 'object', mode: 'optional' });
    if (!expandVariant(path, v.key)) continue;
    const body = mergeBranches(v.branches.map((b) => (u.content ? norm(b.properties[u.content]) : b)));
    runAll(emitObjectBody(body, p, u.content ? undefined : r.property));
  }
  for (const e of u.extra ?? []) {
    const p = join(path, e.name);
    attrs.push({ path: p, kind: e.kind, mode: e.mode });
    members.push(p);
    values[p] = e.values;
  }
  const g: Group = { kind: u.group ?? 'exactly_one_of', members };
  if (u.keep === 'sibling' && u['emit-discriminator'] !== false) g.discriminator = { path: join(path, r.property!), values };
  groups.push(g);
  return [];
};

// ------------------------------------------------------------------ root
loadDoc(input);
const root = norm({ $ref: '#/components/schemas/Kibana_HTTP_APIs_kbn-dashboard-data' });
for (const e of x(root, 'extra') ?? []) attrs.push({ path: e.name, kind: e.kind, mode: e.mode });
for (const f of fieldSpecs(root)) {
  if (f.prop === 'panels') {
    // U1 special case: x-tf-split moves the section branch to a root attribute
    const items = f.schema.items;
    const split = x(items, 'split');
    const [panelUnion, section] = splitBranches(items);
    attrs.push({ path: 'panels', kind: 'list', element: 'object', mode: defaultMode(f) });
    runAll(emitObjectBody(panelUnion, 'panels[]'));
    attrs.push({ path: split.attribute, kind: 'list', element: 'object', mode: defaultMode(f) });
    runAll(emitObjectBody(section, `${split.attribute}[]`));
    continue;
  }
  emitValue(f.schema, f.tfName, defaultMode(f));
}

// manual mode exceptions not expressible on a schema node (kept here so they are counted)
attrs.sort((a, b) => (a.path < b.path ? -1 : a.path > b.path ? 1 : 0));
const dupes = attrs.filter((a, i) => i && attrs[i - 1].path === a.path).map((a) => a.path);
if (dupes.length) console.error('DUPLICATE PATHS', dupes);
writeFileSync(
  output,
  JSON.stringify(
    {
      format: 'attribute-tree/v1',
      resource: 'elasticstack_kibana_dashboard',
      source: `spike F: ${input} (Kibana OAS + provider overlay) via prototype/dashboards_tf_f/src/emit_attribute_tree.ts${FULL ? ' --full' : ''}`,
      attributes: attrs,
      groups,
    },
    null,
    2,
  ),
);
writeFileSync(output.replace(/\.json$/, '.variant-lock.json'), JSON.stringify(lock, null, 2));
for (const n of [...new Set(lint.notes)]) console.error(`NOTE ${n}`);
console.error(`attributes=${attrs.length} groups=${groups.length} unmapped-unions=${lint.unmapped.length}`);
for (const u of lint.unmapped) console.error(`UNMAPPED ${u}`);
void schemas;
