// PROTOTYPE (spike F): shared reader of the overlaid OAS. Interprets the `x-tf-*` vocabulary and turns
// schema nodes into "projection specs" that the three consumers (attribute-tree emitter, value conversion
// API->TF, value conversion TF->API) all walk. Throwaway code: no polish, no tests.
import { readFileSync } from 'node:fs';
import yaml from 'js-yaml';

export type Schema = Record<string, any>;
export type Kind = 'string' | 'number' | 'bool' | 'object' | 'list' | 'set' | 'map' | 'json-string';
export type Mode = 'required' | 'optional' | 'optional_computed' | 'computed';

export interface Recognizer {
  property?: string;
  'when-absent'?: string;
  present?: string;
}
export interface UnionX {
  mode: 'keys' | 'merge';
  recognizer?: Recognizer;
  keep?: 'sibling' | 'none';
  content?: string;
  key?: string;
  variants?: Record<string, string | string[]>;
  'default-variant'?: string;
  exclude?: string[];
  group?: 'exactly_one_of' | 'conflicts_with';
  'emit-discriminator'?: boolean;
  extra?: Array<{ name: string; kind: Kind; mode: Mode; values: string[] }>;
}

export const lint: { unmapped: string[]; notes: string[] } = { unmapped: [], notes: [] };

let DOC: Schema;
export const loadDoc = (file: string): Schema => {
  const text = readFileSync(file, 'utf8');
  DOC = file.endsWith('.json') ? JSON.parse(text) : (yaml.load(text) as Schema);
  return DOC;
};
export const schemas = (): Record<string, Schema> => DOC.components.schemas;

const xKeys = (n: Schema | undefined): Schema =>
  Object.fromEntries(Object.entries(n ?? {}).filter(([k]) => k.startsWith('x-tf-')));
export const x = (n: Schema | undefined, k: string): any => n?.[`x-tf-${k}`];

const refName = (ref: string): string => ref.replace('#/components/schemas/', '');

/** Follows $ref chains. x-tf-* on the usage site override the referenced component's (per-location overrides). */
export const deref = (node: Schema): Schema => {
  let cur = node;
  const usage: Schema[] = [];
  const seen = new Set<string>();
  while (cur && cur.$ref) {
    if (seen.has(cur.$ref)) throw new Error(`ref cycle ${cur.$ref}`);
    seen.add(cur.$ref);
    usage.push(xKeys(cur));
    const target = schemas()[refName(cur.$ref)];
    if (!target) throw new Error(`unresolved ${cur.$ref}`);
    cur = { ...target, 'x-tf-ref': refName(cur.$ref) };
  }
  for (const u of usage.reverse()) cur = { ...cur, ...u };
  return cur;
};

const unionKey = (n: Schema): 'anyOf' | 'oneOf' | undefined =>
  n.anyOf ? 'anyOf' : n.oneOf ? 'oneOf' : undefined;

const isStringEnum = (n: Schema): boolean => n.type === 'string' && Array.isArray(n.enum);

/** Fully normalized node: deref'd, allOf merged/distributed, enum-only unions collapsed. Memoized per object. */
const memo = new WeakMap<object, Schema>();
export const norm = (input: Schema): Schema => {
  if (memo.has(input)) return memo.get(input)!;
  let n = deref(input);
  if (n.allOf) {
    const { allOf, ...rest } = n;
    const parts = (allOf as Schema[]).map(norm);
    n = parts.reduce((acc, p) => mergeAll(acc, p), stripUnionless(rest));
  }
  const uk = unionKey(n);
  if (uk) {
    const branches = (n[uk] as Schema[]).map(norm);
    if (branches.every(isStringEnum)) {
      const { [uk]: _drop, ...rest } = n;
      n = { ...rest, type: 'string', enum: branches.flatMap((b) => b.enum) };
    } else if (branches.length === 1 && !x(n, 'union')) {
      const { [uk]: _drop, ...rest } = n;
      n = mergeAll(stripUnionless(rest), branches[0]);
    } else {
      n = { ...n, [uk]: branches };
    }
  }
  memo.set(input, n);
  return n;
};
const stripUnionless = (n: Schema): Schema => n;

/** allOf semantics: objects merge; a union member is distributed over the other members. */
const mergeAll = (a: Schema, b: Schema): Schema => {
  const ua = unionKey(a);
  const ub = unionKey(b);
  if (ub && !ua) return distribute(b, ub, a);
  if (ua && !ub) return distribute(a, ua, b);
  if (ua && ub) throw new Error('allOf of two unions not supported in prototype');
  const out: Schema = { ...a, ...b };
  if (a.properties || b.properties) out.properties = { ...(a.properties ?? {}), ...(b.properties ?? {}) };
  if (a.required || b.required) out.required = [...new Set([...(a.required ?? []), ...(b.required ?? [])])];
  if (out.properties) out.type = 'object';
  delete out.default;
  return out;
};
const distribute = (u: Schema, uk: 'anyOf' | 'oneOf', other: Schema): Schema => {
  const { properties, required, type, additionalProperties, title, ...otherX } = other;
  const objPart = { properties, required, type };
  return {
    ...xKeys(otherX),
    ...u,
    [uk]: (u[uk] as Schema[]).map((br) => mergeAll(norm(br), objPart)),
  };
};

export const branchesOf = (n: Schema): Schema[] => {
  const uk = unionKey(n);
  return uk ? (n[uk] as Schema[]).map(norm) : [];
};
export const isUnion = (n: Schema): boolean => !!unionKey(n);

// ------------------------------------------------------------------ recognizers

/** Recognizer values a branch can carry (nested unions contribute all their leaves' values). */
export const recognizerValues = (branch: Schema, r: Recognizer): string[] => {
  if (isUnion(branch)) return [...new Set(branchesOf(branch).flatMap((b) => recognizerValues(b, r)))];
  if (r.present) return branch.properties?.[r.present] ? ['present'] : ['absent'];
  const p = branch.properties?.[r.property!];
  if (!p) return r['when-absent'] ? [r['when-absent']] : [];
  const pn = norm(p);
  if (pn.enum) return pn.enum.map(String);
  if (pn.const !== undefined) return [String(pn.const)];
  return r['when-absent'] ? [r['when-absent']] : [];
};

export const recognize = (value: any, r: Recognizer): string | undefined => {
  if (r.present) return value != null && value[r.present] !== undefined ? 'present' : 'absent';
  const v = value?.[r.property!];
  return v === undefined ? r['when-absent'] : String(v);
};

export const keyFor = (u: UnionX, value: string): string | undefined => {
  if (u.exclude?.includes(value)) return undefined;
  for (const [k, vs] of Object.entries(u.variants ?? {})) {
    if ((Array.isArray(vs) ? vs : [vs]).includes(value)) return k;
  }
  if (u['default-variant']) return u['default-variant'];
  return (u.key ?? '{value}').replace('{value}', value);
};

export interface Variant {
  key: string;
  values: string[];
  branches: Schema[]; // one or more API branches projected into this key (merged)
}

/** Groups a union's branches into variant keys (order: first appearance). */
export const variantsOf = (n: Schema, u: UnionX, where: string): Variant[] => {
  const out = new Map<string, Variant>();
  for (const b of branchesOf(n)) {
    const values = recognizerValues(b, u.recognizer!);
    if (!values.length) lint.unmapped.push(`${where}: branch ${b['x-tf-ref'] ?? b.title ?? '?'} has no recognizer value`);
    for (const v of values) {
      const k = keyFor(u, v);
      if (!k) continue;
      const cur = out.get(k) ?? { key: k, values: [], branches: [] };
      if (!cur.values.includes(v)) cur.values.push(v);
      if (!cur.branches.includes(b)) cur.branches.push(b);
      out.set(k, cur);
    }
  }
  return [...out.values()];
};

// ------------------------------------------------------------------ merge (undiscriminated unions with no variant key)

/** Merges several object (or union) schemas property-by-property. Required = required in every branch. */
export const mergeBranches = (bs: Schema[]): Schema => {
  if (bs.length === 1) return bs[0];
  if (bs.every(isUnion)) {
    return { ...pickX(bs), anyOf: bs.flatMap(branchesOf) };
  }
  if (bs.every((b) => b.type === 'array')) {
    return { ...pickX(bs), type: 'array', items: mergeBranches(bs.map((b) => norm(b.items))) };
  }
  if (bs.every((b) => b.properties)) {
    // keyed by API name, split when branches rename the same API property differently (legend position/alignment)
    const props: Record<string, Schema[]> = {};
    const req: Record<string, boolean> = {};
    for (const b of bs)
      for (const [k, v] of Object.entries(b.properties)) {
        const s = norm(v as Schema);
        const tfName = x(s, 'name');
        const mk = tfName && tfName !== k && x(s, 'api-prop') === undefined ? `${k}@${tfName}` : k;
        (props[mk] ??= []).push(mk === k ? s : { ...s, 'x-tf-api-prop': k });
      }
    for (const k of Object.keys(props)) {
      const api = k.split('@')[0];
      req[k] = bs.every((b) => (b.required ?? []).includes(api));
    }
    const required = Object.keys(props).filter((k) => req[k]);
    // lint: a mapping annotation present on one branch but not on another is a likely uncovered API change
    for (const [k, vs] of Object.entries(props)) {
      if (vs.length !== bs.length) continue;
      const sig = vs.map((v) => ['json', 'custom', 'ignore'].filter((n) => x(v, n) || x(v.items, n)).join('+'));
      if (new Set(sig).size > 1)
        lint.notes.push(`merge asymmetry on '${k}': ${bs.map((b, i) => `${b['x-tf-ref'] ?? b.title ?? 'inline'}=[${sig[i]}]`).join(' ')}`);
    }
    const properties = Object.fromEntries(
      Object.entries(props).map(([k, vs]) => [k, vs.length === 1 ? vs[0] : mergeBranches(vs)]),
    );
    return { ...pickX(bs), type: 'object', properties, required };
  }
  return { ...bs[0], ...pickX(bs) };
};
const pickX = (bs: Schema[]): Schema => Object.assign({}, ...[...bs].reverse().map(xKeys));

/** For merge-mode value conversion: first branch whose properties accept every key of the value. */
export const firstFit = (bs: Schema[], value: any, fieldsOf: (b: Schema) => string[]): Schema | undefined =>
  bs.find((b) => Object.keys(value ?? {}).every((k) => fieldsOf(b).includes(k)) && constsMatch(b, value));
const constsMatch = (b: Schema, value: any): boolean =>
  Object.entries(value ?? {}).every(([k, v]) => {
    const p = b.properties?.[k] && norm(b.properties[k]);
    return !p?.enum || typeof v !== 'string' || p.enum.includes(v);
  });

// ------------------------------------------------------------------ field specs (object level)

export interface FieldSpec {
  prop: string; // API property name
  schema: Schema; // normalized
  required: boolean;
  op: 'plain' | 'json' | 'flatten' | 'unwrap';
  tfName: string;
  prefix?: string;
  unwrap?: string;
  hoist?: boolean;
  enumMap?: Record<string, any>;
  mode?: Mode;
}

/** Field specs for an object schema, honoring ignore / rename / json / flatten / unwrap / hoist. */
export const fieldSpecs = (obj: Schema, recognizerToDrop?: string): FieldSpec[] => {
  const out: FieldSpec[] = [];
  const policy = modePolicy.at(-1);
  for (const [key, raw] of Object.entries(obj.properties ?? {})) {
    const s = norm(raw as Schema);
    const prop: string = x(s, 'api-prop') ?? key;
    if (x(s, 'ignore')) continue;
    if (prop === recognizerToDrop && !x(s, 'keep-recognizer')) continue;
    const required = (obj.required ?? []).includes(prop);
    const json = x(s, 'json');
    const flatten = x(s, 'flatten');
    const unwrap = x(s, 'unwrap');
    const base = {
      prop,
      schema: s,
      required,
      hoist: !!x(s, 'hoist'),
      enumMap: x(s, 'enum-map'),
      mode: x(s, 'mode') ?? (s.default !== undefined && !required ? policy : undefined),
    };
    if (json && !json.wrap) out.push({ ...base, op: 'json', tfName: x(s, 'name') ?? json.name ?? `${prop}_json` });
    else if (flatten) out.push({ ...base, op: 'flatten', tfName: prop, prefix: flatten.prefix ?? '' });
    else if (unwrap) out.push({ ...base, op: 'unwrap', tfName: x(s, 'name') ?? prop, unwrap });
    else out.push({ ...base, op: 'plain', tfName: x(s, 'name') ?? prop });
  }
  return out;
};

/** Stack of x-tf-default-mode policies (mode for API-defaulted, non-required fields in a subtree). */
export const modePolicy: Array<Mode | undefined> = [];
export const withPolicy = <T>(s: Schema, fn: () => T): T => {
  const p = x(s, 'default-mode');
  if (!p) return fn();
  modePolicy.push(p);
  try {
    return fn();
  } finally {
    modePolicy.pop();
  }
};

export const defaultMode = (f: { required: boolean; mode?: Mode }): Mode => f.mode ?? (f.required ? 'required' : 'optional');

/** Primitive TF kind for a normalized non-object schema. */
export const primKind = (s: Schema): Kind => {
  if (x(s, 'type')) return x(s, 'type');
  if (s.type === 'boolean') return 'bool';
  if (s.type === 'number' || s.type === 'integer') return 'number';
  if (s.type === 'string' || s.enum) return 'string';
  return 'json-string';
};

/** U1 split: [kept branch, split-out branch], chosen by the x-tf-split recognizer (not by position). */
export const splitBranches = (items: Schema): [Schema, Schema] => {
  const split = x(items, 'split');
  const bs = branchesOf(norm(items));
  const out = bs.filter((b) => recognizerValues(b, split.recognizer).includes('present'));
  const keep = bs.filter((b) => !out.includes(b));
  if (out.length !== 1 || keep.length !== 1) throw new Error('x-tf-split expects exactly one split-out branch');
  return [keep[0], out[0]];
};
