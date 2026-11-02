// PROTOTYPE (spike G): the baseline requires xy `legend` and `styling.fitting.type`, which the Kibana OAS
// examples omit. Simulate "Kibana GET filled them in" by adding minimal values, so the pipeline
// can continue past generation. Writes to prototype_g/fixtures_xy_patched/.
import fs from 'node:fs';
import path from 'node:path';
const [src, dst] = process.argv.slice(2);
fs.mkdirSync(dst, { recursive: true });
const patch = (p) => {
  if (p?.type === 'vis' && p.config?.type === 'xy') {
    p.config.legend ??= { visibility: 'auto', placement: 'outside', position: 'bottom' };
    p.config.styling ??= {};
    p.config.styling.fitting ??= { type: 'none' };
  }
  (p?.panels ?? []).forEach(patch);
};
for (const f of ['oas_create_simple.json', 'oas_sections_and_controls.json']) {
  const d = JSON.parse(fs.readFileSync(path.join(src, f), 'utf8'));
  d.panels.forEach(patch);
  fs.writeFileSync(path.join(dst, f), JSON.stringify(d, null, 2));
}
