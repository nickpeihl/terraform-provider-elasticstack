// PROTOTYPE (spike G): deep-compare fixture vs API JSON the provider wrote, after
// normalizing panels-before-sections. Exit 0 = equal, 1 = diff, 2 = missing.
import fs from 'node:fs';
const [fixturePath, writtenPath] = process.argv.slice(2);
if (!fs.existsSync(writtenPath)) { console.log('NO WRITE CAPTURED'); process.exit(2); }
const isSection = (p) => p && Array.isArray(p.panels);
const norm = (d) => {
  const c = structuredClone(d);
  if (Array.isArray(c.panels)) c.panels = [...c.panels.filter((p) => !isSection(p)), ...c.panels.filter(isSection)];
  return c;
};
const a = norm(JSON.parse(fs.readFileSync(fixturePath, 'utf8')));
const b = norm(JSON.parse(fs.readFileSync(writtenPath, 'utf8')));
const diffs = [];
const walk = (x, y, p) => {
  if (JSON.stringify(x) === JSON.stringify(y)) return;
  if (x && y && typeof x === 'object' && typeof y === 'object' && Array.isArray(x) === Array.isArray(y)) {
    for (const k of new Set([...Object.keys(x), ...Object.keys(y)])) walk(x[k], y[k], `${p}${Array.isArray(x) ? `[${k}]` : `.${k}`}`);
    return;
  }
  diffs.push(`${p}: fixture=${JSON.stringify(x)} written=${JSON.stringify(y)}`);
};
walk(a, b, '$');
console.log(diffs.length ? diffs.join('\n') : 'EQUAL');
process.exit(diffs.length ? 1 : 0);
