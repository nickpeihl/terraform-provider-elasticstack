// PROTOTYPE (spike G): fake Kibana that serves spike0 fixtures as GET /api/dashboards/{id}
// and records every write body. No ES, no real Kibana.
// Usage: node fake_kibana.mjs <fixturesDir> <captureDir> [port]
import http from 'node:http';
import fs from 'node:fs';
import path from 'node:path';

const [fixturesDir, captureDir, portArg] = process.argv.slice(2);
const port = Number(portArg ?? 5699);
const version = process.env.FAKE_KIBANA_VERSION ?? '9.5.0';
fs.mkdirSync(captureDir, { recursive: true });
const store = new Map(); // id -> data (overrides fixture after a write)

const send = (res, code, body) => {
  res.writeHead(code, { 'content-type': 'application/json' });
  res.end(body === undefined ? '' : JSON.stringify(body));
};

const load = (id) => {
  if (store.has(id)) return store.get(id);
  const f = path.join(fixturesDir, `${id}.json`);
  return fs.existsSync(f) ? JSON.parse(fs.readFileSync(f, 'utf8')) : undefined;
};

http
  .createServer((req, res) => {
    let raw = '';
    req.on('data', (c) => (raw += c));
    req.on('end', () => {
      const url = new URL(req.url, 'http://x');
      const p = url.pathname.replace(/^\/s\/[^/]+/, '');
      fs.appendFileSync(path.join(captureDir, 'requests.log'), `${req.method} ${req.url}\n`);
      if (req.method === 'GET' && p === '/api/status') {
        return send(res, 200, { version: { number: version, build_flavor: 'default' } });
      }
      const m = p.match(/^\/api\/dashboards(?:\/([^/]+))?$/);
      if (!m) return send(res, 404, { message: `fake kibana: no route ${req.method} ${p}` });
      let id = m[1] ? decodeURIComponent(m[1]) : undefined;
      if (req.method === 'GET' && id) {
        const data = load(id);
        if (!data) return send(res, 404, { message: 'not found' });
        return send(res, 200, { id, data, meta: { version: '1' } });
      }
      if (req.method === 'POST' || req.method === 'PUT') {
        const body = JSON.parse(raw || '{}');
        id = id ?? `created-${Date.now()}`;
        fs.writeFileSync(path.join(captureDir, `${req.method}-${id}.json`), JSON.stringify(body, null, 2));
        store.set(id, body);
        return send(res, req.method === 'POST' ? 201 : 200, { id, data: body, meta: { version: '1' } });
      }
      if (req.method === 'DELETE') {
        store.delete(id);
        return send(res, 200, {});
      }
      return send(res, 404, { message: 'fake kibana: unsupported' });
    });
  })
  .listen(port, '127.0.0.1', () => console.log(`fake kibana on ${port}`));
