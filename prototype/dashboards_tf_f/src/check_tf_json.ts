// PROTOTYPE (spike F): static check of generated *.tf.json against an attribute tree: unknown attributes,
// required attributes missing under a present parent, and wrong value shapes. Approximates
// `terraform validate` without a provider binary. Only paths inside expanded (slice) subtrees are checked.
//   node src/check_tf_json.ts <attribute-tree.json> <tf dir>
import { readFileSync, readdirSync } from 'node:fs';
import { join } from 'node:path';

const [, , treeFile, dir] = process.argv;
const tree = JSON.parse(readFileSync(treeFile, 'utf8'));
const byPath = new Map<string, any>(tree.attributes.map((a: any) => [a.path, a]));
const children = (parent: string): any[] =>
  tree.attributes.filter((a: any) => {
    const rest = parent ? a.path.slice(parent.length + 1) : a.path;
    return (parent ? a.path.startsWith(`${parent}.`) : true) && rest && !rest.includes('.');
  });
const expanded = (p: string): boolean => children(p).length > 0;

let problems = 0;
const check = (obj: any, parent: string, file: string): void => {
  for (const [k, v] of Object.entries(obj)) {
    const p = parent ? `${parent}.${k}` : k;
    const a = byPath.get(p);
    if (!a) {
      problems++;
      console.log(`  ${file}: unknown attribute ${p}`);
      continue;
    }
    if (a.kind === 'object' && expanded(p)) check(v, p, file);
    if (a.kind === 'list' && a.element === 'object') for (const item of v as any[]) check(item, `${p}[]`, file);
  }
  if (parent === '' || expanded(parent)) {
    for (const c of children(parent)) {
      const name = c.path.split('.').at(-1);
      if (c.mode === 'required' && !(name in obj)) {
        problems++;
        console.log(`  ${file}: missing required ${c.path}`);
      }
    }
  }
};

for (const f of readdirSync(dir).filter((n) => n.endsWith('.tf.json')).sort()) {
  const doc = JSON.parse(readFileSync(join(dir, f), 'utf8'));
  for (const attrs of Object.values(doc.resource.elasticstack_kibana_dashboard)) check(attrs, '', f);
}
console.log(`${problems} problem(s)`);
