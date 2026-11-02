// PROTOTYPE (spike F): value conversion of every spike0 fixture to *.tf.json, then the round trip
// fixture -> Terraform-shaped JSON -> API JSON, deep-compared after normalizing panels-before-sections.
//   node src/convert_and_round_trip.ts <overlaid.yaml> <fixtures dir> <out dir>
import { mkdirSync, readFileSync, readdirSync, writeFileSync } from 'node:fs';
import { basename, join } from 'node:path';
import { isDeepStrictEqual } from 'node:util';
import { loadDoc, norm } from './f_core.ts';
import { apiToTfRoot, tfToApiRoot } from './value_conversion.ts';

const [, , input, fixturesDir, outDir] = process.argv;
loadDoc(input);
const root = norm({ $ref: '#/components/schemas/Kibana_HTTP_APIs_kbn-dashboard-data' });
mkdirSync(outDir, { recursive: true });

// Terraform template escaping for *.tf.json string values
const mapStrings = (v: any, fn: (s: string) => string): any =>
  typeof v === 'string' ? fn(v) : Array.isArray(v) ? v.map((i) => mapStrings(i, fn)) : v && typeof v === 'object'
    ? Object.fromEntries(Object.entries(v).map(([k, i]) => [k, mapStrings(i, fn)]))
    : v;
// split/join, not replaceAll: '$$' in a string replacement pattern means a single '$'
const escape = (s: string): string => s.split('${').join('$${').split('%{').join('%%{');
const unescape = (s: string): string => s.split('$${').join('${').split('%%{').join('%{');

const normalize = (d: any): any =>
  d.panels
    ? { ...d, panels: [...d.panels.filter((p: any) => !Array.isArray(p.panels)), ...d.panels.filter((p: any) => Array.isArray(p.panels))] }
    : d;

const diffs = (a: any, b: any, path = '$'): string[] => {
  if (isDeepStrictEqual(a, b)) return [];
  if (a && b && typeof a === 'object' && typeof b === 'object' && Array.isArray(a) === Array.isArray(b)) {
    const keys = new Set([...Object.keys(a), ...Object.keys(b)]);
    return [...keys].flatMap((k) => diffs(a[k], b[k], `${path}.${k}`));
  }
  return [`${path}: fixture=${JSON.stringify(a)} roundtrip=${JSON.stringify(b)}`];
};

let failed = 0;
const files = readdirSync(fixturesDir).filter((f) => f.endsWith('.json') && f !== 'index.json').sort();
for (const file of files) {
  const name = basename(file, '.json');
  const fixture = JSON.parse(readFileSync(join(fixturesDir, file), 'utf8'));
  let line = `${name}: `;
  try {
    const attrs = mapStrings(apiToTfRoot(root, fixture), escape);
    const tfJson = { resource: { elasticstack_kibana_dashboard: { [name]: attrs } } };
    writeFileSync(join(outDir, `${name}.tf.json`), JSON.stringify(tfJson, null, 2) + '\n');
    const back = tfToApiRoot(root, mapStrings(tfJson.resource.elasticstack_kibana_dashboard[name], unescape));
    writeFileSync(join(outDir, `${name}.roundtrip.json`), JSON.stringify(back, null, 2) + '\n');
    const d = diffs(normalize(fixture), normalize(back));
    if (d.length) {
      failed++;
      line += `FAIL\n    ${d.join('\n    ')}`;
    } else line += 'PASS';
  } catch (e) {
    failed++;
    line += `ERROR ${(e as Error).message}`;
  }
  console.log(line);
}
console.log(`${files.length - failed}/${files.length} fixtures round-trip losslessly`);
process.exitCode = failed ? 1 : 0;
