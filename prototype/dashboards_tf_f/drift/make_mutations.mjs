// PROTOTYPE (spike F): synthetic Kibana re-bundling mutations of the pinned OAS, to probe overlay brittleness.
//   node drift/make_mutations.mjs <pinned.json> <outdir>
import { readFileSync, writeFileSync, mkdirSync } from 'node:fs';
const [, , src, outDir] = process.argv;
mkdirSync(outDir, { recursive: true });
const load = () => JSON.parse(readFileSync(src, 'utf8'));
const P = 'Kibana_HTTP_APIs_';
const walk = (n, fn) => {
  if (Array.isArray(n)) return n.forEach((i) => walk(i, fn));
  if (n && typeof n === 'object') {
    fn(n);
    Object.values(n).forEach((v) => walk(v, fn));
  }
};
const renameRefs = (doc, from, to) => {
  doc.components.schemas[to] = doc.components.schemas[from];
  delete doc.components.schemas[from];
  walk(doc, (n) => {
    if (n.$ref === `#/components/schemas/${from}`) n.$ref = `#/components/schemas/${to}`;
    if (n.mapping) for (const k of Object.keys(n.mapping)) if (n.mapping[k] === `#/components/schemas/${from}`) n.mapping[k] = `#/components/schemas/${to}`;
  });
};
const save = (name, doc) => writeFileSync(`${outDir}/${name}.json`, JSON.stringify(doc));

// M1: every anyOf/oneOf array reversed (bundler or Zod reorders union members)
{
  const d = load();
  walk(d.components.schemas, (n) => ['anyOf', 'oneOf'].forEach((k) => Array.isArray(n[k]) && n[k].reverse()));
  save('m1_reordered_unions', d);
}
// M2: a component referenced by a target is renamed (Kibana renames a Zod schema id)
{
  const d = load();
  renameRefs(d, `${P}visXyStylingBars`, `${P}visXyBarsStyling`);
  save('m2_renamed_component', d);
}
// M3: the bundler extracts the (previously inline) by-reference drilldown union into a shared component
//     that the by-value chrome and the by-reference branch both reference.
{
  const d = load();
  const vis = d.components.schemas[`${P}kbn-dashboard-panel-type-vis`].properties.config.anyOf;
  const byRef = vis.find((b) => b.properties?.ref_id);
  const byVal = vis.find((b) => b.allOf).allOf.find((m) => m.properties?.drilldowns);
  d.components.schemas[`${P}visDrilldownItem`] = byRef.properties.drilldowns.items;
  byRef.properties.drilldowns.items = { $ref: `#/components/schemas/${P}visDrilldownItem` };
  byVal.properties.drilldowns.items = { $ref: `#/components/schemas/${P}visDrilldownItem` };
  save('m3_shared_drilldowns', d);
}
// M4: a component that a target names is inlined at its single use site (the reverse of M3)
{
  const d = load();
  const fit = d.components.schemas[`${P}visXyFitting`];
  walk(d.components.schemas, (n) => {
    for (const [k, v] of Object.entries(n)) if (v && v.$ref === `#/components/schemas/${P}visXyFitting`) n[k] = structuredClone(fit);
  });
  delete d.components.schemas[`${P}visXyFitting`];
  save('m4_inlined_component', d);
}
