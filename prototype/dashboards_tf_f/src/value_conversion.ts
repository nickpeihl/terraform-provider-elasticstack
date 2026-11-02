// PROTOTYPE (spike F): value conversion driven by the same overlaid OAS + x-tf-* vocabulary as the
// schema emitter. apiToTf: dashboard API JSON -> Terraform-shaped JSON (attribute values of one
// elasticstack_kibana_dashboard). tfToApi: the inverse, used by the round-trip check.
import {
  type Schema,
  type UnionX,
  branchesOf,
  fieldSpecs,
  isUnion,
  keyFor,
  mergeBranches,
  norm,
  recognize,
  splitBranches,
  variantsOf,
  x,
} from './f_core.ts';

type Obj = Record<string, any>;

// Named hand-written codecs referenced by x-tf-custom (none exercised by the fixtures).
const CODECS: Record<string, { toTf: (v: any) => any; toApi: (v: any) => any }> = {
  xy_reference_line_thresholds: {
    toTf: () => {
      throw new Error('codec xy_reference_line_thresholds not implemented in the prototype');
    },
    toApi: () => {
      throw new Error('codec xy_reference_line_thresholds not implemented in the prototype');
    },
  },
};

const mapEnum = (m: Obj | undefined, v: any): any => (m && v in m ? m[v] : v);
const unmapEnum = (m: Obj | undefined, v: any): any => {
  if (!m) return v;
  const hit = Object.entries(m).find(([, tv]) => tv === v);
  return hit ? hit[0] : v;
};

// ------------------------------------------------------------------ API -> TF

export const apiToTfRoot = (root: Schema, api: Obj): Obj => {
  const out: Obj = {};
  for (const f of fieldSpecs(root)) {
    if (!(f.prop in api)) continue;
    if (f.prop === 'panels') {
      const split = x(f.schema.items, 'split');
      const [panelUnion, section] = splitBranches(f.schema.items);
      const isSection = (p: Obj): boolean => recognize(p, split.recognizer) === 'present';
      const panels = api.panels.filter((p: Obj) => !isSection(p));
      const sections = api.panels.filter(isSection);
      if (panels.length) out.panels = panels.map((p: Obj) => objToTf(panelUnion, p, undefined, {}));
      if (sections.length) out[split.attribute] = sections.map((p: Obj) => objToTf(section, p, undefined, {}));
      continue;
    }
    setField(out, f, api[f.prop], out);
  }
  return out;
};

const setField = (out: Obj, f: ReturnType<typeof fieldSpecs>[number], val: any, sink: Obj): void => {
  const target = f.hoist ? sink : out;
  if (val === undefined) return;
  if (f.op === 'json') target[f.tfName] = JSON.stringify(val);
  else if (f.op === 'unwrap') {
    if (val?.[f.unwrap!] !== undefined) target[f.tfName] = mapEnum(f.enumMap, val[f.unwrap!]);
  } else target[f.tfName] = valueToTf(f.schema, val, out, f.enumMap);
};

const objToTf = (s: Schema, v: Obj, drop: string | undefined, sink: Obj, prefix = ''): Obj => {
  if (isUnion(s)) return unionToTf(s, v, drop, sink);
  const out: Obj = {};
  for (const f of fieldSpecs(s, drop)) {
    if (!(f.prop in v)) continue;
    if (f.op === 'flatten') {
      Object.assign(out, objToTf(f.schema, v[f.prop], undefined, sink, prefix + (f.prefix ?? '')));
      continue;
    }
    setField(out, { ...f, tfName: prefix + f.tfName }, v[f.prop], sink);
  }
  return out;
};

const valueToTf = (s: Schema, v: any, sink: Obj, enumMap?: Obj): any => {
  if (v === null) return null;
  const custom = x(s, 'custom');
  if (custom) return CODECS[custom.codec].toTf(v);
  if (x(s, 'json') && !x(s, 'json').wrap) return JSON.stringify(v);
  if (s.type === 'array') {
    const items = norm(s.items);
    const wrap = x(items, 'json')?.wrap;
    if (wrap) return v.map((i: any) => ({ [wrap]: JSON.stringify(i) }));
    if (isUnion(items) || items.properties) return v.map((i: Obj) => objToTf(items, i, undefined, {}));
    return v.map((i: any) => mapEnum(enumMap, i));
  }
  if (isUnion(s) || s.properties) return objToTf(s, v, undefined, sink);
  return mapEnum(enumMap, v);
};

const siblingSchema = (first: Schema, u: UnionX): Schema => {
  const props: Obj = {};
  for (const [k, v] of Object.entries(first.properties ?? {})) {
    if (k === u.content) continue;
    if (k === u.recognizer!.property && u.keep !== 'sibling') continue;
    if (!u.content && k !== u.recognizer!.property) continue;
    props[k] = k === u.recognizer!.property ? { type: 'string' } : v;
  }
  return { properties: props, required: first.required ?? [] };
};

const variantBody = (u: UnionX, branches: Schema[]): Schema =>
  mergeBranches(branches.map((b) => (u.content ? norm(b.properties[u.content]) : b)));

const unionToTf = (s: Schema, v: Obj, drop: string | undefined, sink: Obj): Obj => {
  const u: UnionX | undefined = x(s, 'union');
  if (!u) throw new Error(`unmapped union (${branchesOf(s).map((b) => b['x-tf-ref'] ?? b.title).join('|')})`);
  if (u.mode === 'merge') {
    const b = branchesOf(s).find((br) => apiFits(br, v)) ?? mergeBranches(branchesOf(s));
    return objToTf(b, v, drop, sink);
  }
  const value = recognize(v, u.recognizer!);
  const key = value === undefined ? undefined : keyFor(u, value);
  const variant = variantsOf(s, u, '').find((vv) => vv.key === key);
  if (!variant) throw new Error(`no variant key for recognizer value ${value}`);
  const out: Obj = {};
  if (u.content || u.keep === 'sibling') Object.assign(out, objToTf(siblingSchema(variant.branches[0], u), v, undefined, out));
  const bodyVal = u.content ? v[u.content] : v;
  out[variant.key] = objToTf(variantBody(u, variant.branches), bodyVal, u.content ? undefined : u.recognizer!.property, out);
  return out;
};

const apiFits = (b: Schema, v: Obj): boolean =>
  !isUnion(b) &&
  Object.keys(v).every((k) => b.properties?.[k]) &&
  Object.entries(v).every(([k, val]) => {
    const p = norm(b.properties[k]);
    return !p.enum || p.enum.includes(val);
  });

// ------------------------------------------------------------------ TF -> API

export const tfToApiRoot = (root: Schema, tf: Obj): Obj => {
  const out: Obj = {};
  for (const f of fieldSpecs(root)) {
    if (f.prop === 'panels') {
      const split = x(f.schema.items, 'split');
      const [panelUnion, section] = splitBranches(f.schema.items);
      const panels = [
        ...(tf.panels ?? []).map((p: Obj) => objToApi(panelUnion, p, undefined, {})),
        ...(tf[split.attribute] ?? []).map((p: Obj) => objToApi(section, p, undefined, {})),
      ];
      if (panels.length) out.panels = panels;
      continue;
    }
    getField(out, f, tf, tf);
  }
  return out;
};

const getField = (out: Obj, f: ReturnType<typeof fieldSpecs>[number], tf: Obj, parentTf: Obj): void => {
  const src = f.hoist ? parentTf : tf;
  if (!src || !(f.tfName in src) || src[f.tfName] === undefined) return;
  const tv = src[f.tfName];
  if (f.op === 'json') out[f.prop] = JSON.parse(tv);
  else if (f.op === 'unwrap') out[f.prop] = { [f.unwrap!]: unmapEnum(f.enumMap, tv) };
  else out[f.prop] = valueToApi(f.schema, tv, tf, f.enumMap);
};

const objToApi = (s: Schema, tf: Obj, drop: string | undefined, parentTf: Obj, prefix = ''): Obj => {
  if (isUnion(s)) return unionToApi(s, tf, drop, parentTf);
  const out: Obj = {};
  for (const f of fieldSpecs(s, drop)) {
    if (f.op === 'flatten') {
      const sub = objToApi(f.schema, tf, undefined, parentTf, prefix + (f.prefix ?? ''));
      if (Object.keys(sub).length) out[f.prop] = sub;
      continue;
    }
    getField(out, { ...f, tfName: prefix + f.tfName }, tf, parentTf);
  }
  return out;
};

const valueToApi = (s: Schema, tv: any, parentTf: Obj, enumMap?: Obj): any => {
  if (tv === null) return null;
  const custom = x(s, 'custom');
  if (custom) return CODECS[custom.codec].toApi(tv);
  if (x(s, 'json') && !x(s, 'json').wrap) return JSON.parse(tv);
  if (s.type === 'array') {
    const items = norm(s.items);
    const wrap = x(items, 'json')?.wrap;
    if (wrap) return tv.map((i: Obj) => JSON.parse(i[wrap]));
    if (isUnion(items) || items.properties) return tv.map((i: Obj) => objToApi(items, i, undefined, {}));
    return tv.map((i: any) => unmapEnum(enumMap, i));
  }
  if (isUnion(s) || s.properties) return objToApi(s, tv, undefined, parentTf);
  return unmapEnum(enumMap, tv);
};

const tfNames = (s: Schema, drop?: string, prefix = ''): string[] =>
  fieldSpecs(s, drop).flatMap((f) => (f.op === 'flatten' ? tfNames(f.schema, undefined, prefix + (f.prefix ?? '')) : [prefix + f.tfName]));

const tfFits = (b: Schema, tf: Obj, drop?: string): boolean => {
  if (isUnion(b)) return false;
  const names = tfNames(b, drop);
  if (!Object.keys(tf).every((k) => names.includes(k))) return false;
  return fieldSpecs(b, drop).every((f) => {
    if (!f.enumMap || !(f.tfName in tf)) return true;
    const allowed = (f.schema.enum ?? Object.keys(f.enumMap)).map((e: string) => f.enumMap![e]);
    return allowed.includes(tf[f.tfName]);
  });
};

const unionToApi = (s: Schema, tf: Obj, drop: string | undefined, parentTf: Obj): Obj => {
  const u: UnionX | undefined = x(s, 'union');
  if (!u) throw new Error('unmapped union');
  if (u.mode === 'merge') {
    const b = branchesOf(s).find((br) => tfFits(br, tf, drop)) ?? mergeBranches(branchesOf(s));
    return objToApi(b, tf, drop, parentTf);
  }
  const variants = variantsOf(s, u, '');
  const variant = variants.find((vv) => tf[vv.key] != null);
  if (!variant) throw new Error(`no variant key set among ${variants.map((v) => v.key)}`);
  const r = u.recognizer!;
  const out: Obj = {};
  if (u.content || u.keep === 'sibling') Object.assign(out, objToApi(siblingSchema(variant.branches[0], u), tf, undefined, parentTf));
  const body = objToApi(variantBody(u, variant.branches), tf[variant.key], u.content ? undefined : r.property, tf);
  if (u.content) out[u.content] = body;
  else Object.assign(out, body);
  if (r.property && u.keep !== 'sibling') {
    if (variant.values.length !== 1) throw new Error(`variant ${variant.key} has several recognizer values; keep it as a sibling`);
    out[r.property] = variant.values[0];
  }
  return out;
};
