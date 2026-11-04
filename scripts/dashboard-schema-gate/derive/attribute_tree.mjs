// Prints the attribute tree Kibana derives from its published OpenAPI document, using Kibana's own
// `buildAttributeTree` (package @kbn/as-code-dashboard-terraform), so the provider compares against exactly what
// Kibana's exporter produces.
//
// Usage: node --import ./resolve_ts.mjs attribute_tree.mjs <kibana package src dir> <schemas.json>
// The JSON document needs only `components.schemas` of `oas_docs/output/kibana.yaml`.
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { pathToFileURL } from 'node:url';

const [srcDir, oasPath] = process.argv.slice(2);
if (!srcDir || !oasPath) {
  console.error('usage: attribute_tree.mjs <kibana package src dir> <schemas.json>');
  process.exit(2);
}
const { buildAttributeTree } = await import(pathToFileURL(join(srcDir, 'attribute_tree.ts')).href);
const doc = JSON.parse(readFileSync(oasPath, 'utf8'));
process.stdout.write(`${JSON.stringify(buildAttributeTree(doc), null, 2)}\n`);
