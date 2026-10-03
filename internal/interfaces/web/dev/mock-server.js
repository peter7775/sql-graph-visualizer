#!/usr/bin/env node
/*
 * Development-only mock of the live benchmark API, so the UI can be built and
 * tested without the Go backend / MySQL.
 *
 *   node internal/interfaces/web/dev/mock-server.js
 *   open http://localhost:3999/
 *
 * Env: PORT (default 3999), TICK_MS (sample interval, default 1000; lower it
 * for faster headless tests).
 *
 * Serves static files from internal/interfaces/web and implements the contract:
 *   GET  /api/demo/scenarios
 *   POST /api/demo/scenarios/{name}/run
 *   GET  /api/demo/topology
 *   GET  /api/demo/optimization, POST /api/demo/optimization/{apply|revert}
 *   POST /api/performance/benchmarks/{id}/stop
 *   GET  /api/performance/benchmarks/compare?a=&b=
 *   GET  /api/performance/benchmarks/{id}/stream   (SSE, replay + live)
 *   GET  /api/performance/benchmarks/{id}/samples?since=
 */
'use strict';

const http = require('http');
const fs = require('fs');
const path = require('path');
const { randomUUID } = require('crypto');

const PORT = Number(process.env.PORT) || 3999;
const TICK_MS = Number(process.env.TICK_MS) || 1000;
const WEB_ROOT = path.resolve(__dirname, '..');

// ---------------------------------------------------------------------------
// Synthetic data
// ---------------------------------------------------------------------------

const TOPOLOGY = {
  nodes: [
    { id: 'customers', table: 'customers', label: 'Customers' },
    { id: 'products', table: 'products', label: 'Products' },
    { id: 'categories', table: 'categories', label: 'Categories' },
    { id: 'orders', table: 'orders', label: 'Orders' },
    { id: 'order_items', table: 'order_items', label: 'Order items' },
  ],
  edges: [
    { id: 'orders-customers', source: 'orders', target: 'customers', relation: 'orders.customer_id -> customers.id' },
    { id: 'order_items-orders', source: 'order_items', target: 'orders', relation: 'order_items.order_id -> orders.id' },
    { id: 'order_items-products', source: 'order_items', target: 'products', relation: 'order_items.product_id -> products.id' },
    { id: 'products-categories', source: 'products', target: 'categories', relation: 'products.category_id -> categories.id' },
  ],
};

const SCENARIOS = [
  { name: 'checkout-peak', title: 'Checkout peak', description: 'Customers browse their orders and checkout lines under peak load.', threads: 16, duration_seconds: 30 },
  { name: 'catalog-browse', title: 'Catalog browsing', description: 'Read-heavy product catalog traffic.', threads: 8, duration_seconds: 20 },
  { name: 'mixed-oltp', title: 'Mixed OLTP', description: 'Reads with a steady stream of order inserts.', threads: 24, duration_seconds: 30 },
];

// weight = share of traffic; before/after = base avg latency in ms without / with the index.
const QUERIES = [
  { pattern: 'SELECT * FROM orders WHERE customer_id = ? ORDER BY created_at DESC LIMIT 20', description: 'Orders by customer', type: 'SELECT', tables: ['orders', 'customers'], edge: 'orders-customers', weight: 0.28, before: 340, after: 2.4 },
  { pattern: 'SELECT o.id, SUM(oi.price * oi.quantity) FROM orders o JOIN order_items oi ON oi.order_id = o.id WHERE o.customer_id = ? GROUP BY o.id', description: 'Customer order totals', type: 'SELECT', tables: ['orders', 'order_items'], edge: 'order_items-orders', weight: 0.2, before: 520, after: 4.1 },
  { pattern: 'SELECT p.* FROM products p JOIN categories c ON c.id = p.category_id WHERE c.id = ?', description: 'Products in category', type: 'SELECT', tables: ['products', 'categories'], edge: 'products-categories', weight: 0.22, before: 38, after: 1.9 },
  { pattern: 'SELECT p.name, oi.quantity FROM order_items oi JOIN products p ON p.id = oi.product_id WHERE oi.order_id = ?', description: 'Order lines with products', type: 'SELECT', tables: ['order_items', 'products'], edge: 'order_items-products', weight: 0.2, before: 120, after: 3.0 },
  { pattern: 'INSERT INTO orders (customer_id, status, created_at) VALUES (?, ?, NOW())', description: 'Create order', type: 'INSERT', tables: ['orders'], edge: null, weight: 0.1, before: 9, after: 6 },
];

const OPTIMIZATION = {
  name: 'idx_orders_customer',
  title: 'Index orders.customer_id',
  description: 'Orders are looked up by customer_id, but the column has no index, so every lookup scans the whole table.',
  statements: ['CREATE INDEX idx_orders_customer_id ON orders (customer_id)'],
  applied: false,
};

const benchmarks = new Map();

// ---------------------------------------------------------------------------
// Sample generation
// ---------------------------------------------------------------------------

const latHeat = piecewise([0, 50, 200, 500], [0, 0.3, 0.8, 1]);

function piecewise(xs, ys) {
  return (x) => {
    if (x <= xs[0]) return ys[0];
    for (let i = 1; i < xs.length; i++) {
      if (x <= xs[i]) {
        const t = (x - xs[i - 1]) / (xs[i] - xs[i - 1]);
        return ys[i - 1] + t * (ys[i] - ys[i - 1]);
      }
    }
    return ys[ys.length - 1];
  };
}

function rank(ms) {
  return ms < 50 ? 'FAST' : ms < 200 ? 'MEDIUM' : 'SLOW';
}

function jitter(amount) {
  return 1 + (Math.random() * 2 - 1) * amount;
}

function round(v, d = 2) {
  const f = Math.pow(10, d);
  return Math.round(v * f) / f;
}

function buildSample(b, seq) {
  const elapsed = seq;
  const ramp = Math.min(1, 0.3 + 0.25 * elapsed);
  const lats = QUERIES.map((q) => (b.indexed ? q.after : q.before) * jitter(0.08) * (0.85 + 0.15 * ramp));
  const meanLat = QUERIES.reduce((s, q, i) => s + q.weight * lats[i], 0);
  const totalQps = ((b.threads * 1000) / meanLat) * 0.9 * ramp * jitter(0.05);

  const queries = QUERIES.map((q, i) => {
    const qps = totalQps * q.weight * jitter(0.06);
    return {
      pattern: q.pattern,
      description: q.description,
      type: q.type,
      count: Math.round(qps),
      qps: round(qps, 1),
      avg_latency_ms: round(lats[i], 2),
      p95_latency_ms: round(lats[i] * 1.7, 2),
      tables: q.tables,
    };
  });

  const totalCount = queries.reduce((s, q) => s + q.count, 0) || 1;
  const sorted = queries.slice().sort((a, b2) => a.avg_latency_ms - b2.avg_latency_ms);
  const at = (frac) => {
    let cum = 0;
    for (const q of sorted) {
      cum += q.count / totalCount;
      if (cum >= frac) return q.avg_latency_ms;
    }
    return sorted[sorted.length - 1].avg_latency_ms;
  };
  const avg = queries.reduce((s, q) => s + q.avg_latency_ms * q.count, 0) / totalCount;
  const p50 = at(0.5) * 0.8;
  const p95 = at(0.95) * 1.5;
  const p99 = at(0.99) * 2.1;
  const errors = b.indexed ? 0 : Math.random() < 0.35 ? Math.ceil(Math.random() * 2) : 0;

  const nodes = TOPOLOGY.nodes.map((n) => {
    const qs = queries.filter((q) => q.tables.includes(n.table));
    const cnt = qs.reduce((s, q) => s + q.count, 0);
    const qps = qs.reduce((s, q) => s + q.qps, 0);
    const lat = cnt ? qs.reduce((s, q) => s + q.avg_latency_ms * q.count, 0) / cnt : 0;
    return { id: n.id, table: n.table, qps: round(qps, 1), avg_latency_ms: round(lat, 2), heat: round(latHeat(lat), 3), queries: cnt };
  });

  const edges = TOPOLOGY.edges.map((e) => {
    const idxs = QUERIES.map((q, i) => (q.edge === e.id ? i : -1)).filter((i) => i >= 0);
    const qps = idxs.reduce((s, i) => s + queries[i].qps, 0);
    const lat = idxs.length ? idxs.reduce((s, i) => s + queries[i].avg_latency_ms, 0) / idxs.length : 0;
    return { id: e.id, source: e.source, target: e.target, qps: round(qps, 1), avg_latency_ms: round(lat, 2), load: round(latHeat(lat), 3), rank: rank(lat) };
  });

  return {
    benchmark_id: b.id,
    seq,
    timestamp: new Date().toISOString(),
    elapsed_seconds: elapsed,
    threads: b.threads,
    interval: {
      queries: totalCount,
      qps: round(totalQps, 1),
      avg_latency_ms: round(avg, 2),
      p50_latency_ms: round(p50, 2),
      p95_latency_ms: round(p95, 2),
      p99_latency_ms: round(p99, 2),
      max_latency_ms: round(p99 * 1.6, 2),
      errors,
    },
    queries,
    graph: { nodes, edges },
  };
}

function summarize(b) {
  const samples = b.samples;
  if (!samples.length) {
    return { qps: 0, avg_latency_ms: 0, p50_latency_ms: 0, p95_latency_ms: 0, p99_latency_ms: 0, error_rate: 0, total_queries: 0, duration_seconds: 0 };
  }
  const total = samples.reduce((s, x) => s + x.interval.queries, 0);
  const errors = samples.reduce((s, x) => s + x.interval.errors, 0);
  const w = (key) => samples.reduce((s, x) => s + x.interval[key] * x.interval.queries, 0) / total;
  const duration = samples[samples.length - 1].elapsed_seconds;
  return {
    qps: round(total / duration, 1),
    avg_latency_ms: round(w('avg_latency_ms'), 2),
    p50_latency_ms: round(w('p50_latency_ms'), 2),
    p95_latency_ms: round(w('p95_latency_ms'), 2),
    p99_latency_ms: round(w('p99_latency_ms'), 2),
    error_rate: round(errors / total, 4),
    total_queries: total,
    duration_seconds: duration,
  };
}

// ---------------------------------------------------------------------------
// Benchmark lifecycle + SSE
// ---------------------------------------------------------------------------

function sse(res, event, data, id) {
  let out = '';
  if (id !== undefined) out += `id: ${id}\n`;
  out += `event: ${event}\ndata: ${JSON.stringify(data)}\n\n`;
  res.write(out);
}

function startBenchmark(scenario, threads, duration) {
  const b = {
    id: randomUUID(),
    scenario: scenario.name,
    threads,
    duration,
    indexed: OPTIMIZATION.applied,
    start_time: new Date().toISOString(),
    samples: [],
    status: 'running',
    summary: null,
    error: '',
    subs: new Set(),
    timer: null,
  };
  benchmarks.set(b.id, b);
  let seq = 0;
  b.timer = setInterval(() => {
    seq += 1;
    const sample = buildSample(b, seq);
    b.samples.push(sample);
    for (const res of b.subs) sse(res, 'sample', sample, sample.seq);
    if (seq >= duration) finishBenchmark(b, 'completed');
  }, TICK_MS);
  return b;
}

function finishBenchmark(b, status) {
  if (b.status !== 'running') return;
  clearInterval(b.timer);
  b.status = status;
  b.summary = summarize(b);
  for (const res of b.subs) {
    sendTerminal(res, b);
    res.end();
  }
  b.subs.clear();
}

function sendTerminal(res, b) {
  sse(res, 'status', { id: b.id, status: b.status, phase: b.status });
  sse(res, 'done', { id: b.id, status: b.status, error: b.error, summary: b.summary });
}

function handleStream(req, res, b, query) {
  let since = Number(query.get('since')) || 0;
  const last = Number(req.headers['last-event-id']);
  if (last > since) since = last;

  res.writeHead(200, {
    'Content-Type': 'text/event-stream',
    'Cache-Control': 'no-cache',
    Connection: 'keep-alive',
    'X-Accel-Buffering': 'no',
  });
  res.write('retry: 2000\n\n');
  for (const s of b.samples) if (s.seq > since) sse(res, 'sample', s, s.seq);

  if (b.status !== 'running') {
    sendTerminal(res, b);
    res.end();
    return;
  }
  sse(res, 'status', { id: b.id, status: 'running', phase: 'load' });
  b.subs.add(res);
  const hb = setInterval(() => res.write(': ping\n\n'), 15000);
  req.on('close', () => {
    clearInterval(hb);
    b.subs.delete(res);
  });
}

// ---------------------------------------------------------------------------
// HTTP plumbing
// ---------------------------------------------------------------------------

function sendJSON(res, status, body) {
  const payload = JSON.stringify(body);
  res.writeHead(status, { 'Content-Type': 'application/json', 'Content-Length': Buffer.byteLength(payload) });
  res.end(payload);
}

const ok = (res, data) => sendJSON(res, 200, { success: true, data, timestamp: new Date().toISOString() });
const fail = (res, status, code, message) =>
  sendJSON(res, status, { success: false, error: { code, message }, timestamp: new Date().toISOString() });

function readBody(req) {
  return new Promise((resolve) => {
    let raw = '';
    req.on('data', (c) => (raw += c));
    req.on('end', () => {
      try {
        resolve(raw ? JSON.parse(raw) : {});
      } catch (e) {
        resolve(null);
      }
    });
  });
}

const MIME = {
  '.html': 'text/html; charset=utf-8',
  '.js': 'application/javascript; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.svg': 'image/svg+xml',
  '.json': 'application/json',
  '.png': 'image/png',
};

function serveFile(res, file) {
  const resolved = path.resolve(file);
  if (!resolved.startsWith(WEB_ROOT + path.sep)) return fail(res, 403, 'forbidden', 'Forbidden');
  fs.readFile(resolved, (err, buf) => {
    if (err) return fail(res, 404, 'not_found', 'Not found');
    res.writeHead(200, { 'Content-Type': MIME[path.extname(resolved)] || 'application/octet-stream', 'Cache-Control': 'no-store' });
    res.end(buf);
  });
}

function pct(a, b) {
  if (!a) return 0;
  return round(((b - a) / a) * 100, 1);
}

function statsOf(b) {
  const s = b.summary;
  return {
    id: b.id,
    scenario: b.scenario,
    qps: s.qps,
    avg_latency_ms: s.avg_latency_ms,
    p95_latency_ms: s.p95_latency_ms,
    p99_latency_ms: s.p99_latency_ms,
    error_rate: s.error_rate,
    total_queries: s.total_queries,
  };
}

const server = http.createServer(async (req, res) => {
  const u = new URL(req.url, 'http://localhost');
  const p = u.pathname;
  const m = req.method;
  let match;

  try {
    // --- API ---
    if (m === 'GET' && p === '/api/demo/scenarios') return ok(res, SCENARIOS);
    if (m === 'GET' && p === '/api/demo/topology') return ok(res, TOPOLOGY);
    if (m === 'GET' && p === '/api/demo/optimization') return ok(res, OPTIMIZATION);

    if (m === 'POST' && (p === '/api/demo/optimization/apply' || p === '/api/demo/optimization/revert')) {
      const apply = p.endsWith('apply');
      await new Promise((r) => setTimeout(r, 600));
      OPTIMIZATION.applied = apply;
      return ok(res, OPTIMIZATION);
    }

    if (m === 'POST' && (match = p.match(/^\/api\/demo\/scenarios\/([^/]+)\/run$/))) {
      const scenario = SCENARIOS.find((s) => s.name === decodeURIComponent(match[1]));
      if (!scenario) return fail(res, 404, 'scenario_not_found', 'Unknown scenario');
      const body = await readBody(req);
      if (body === null) return fail(res, 400, 'bad_request', 'Invalid JSON body');
      for (const b of benchmarks.values()) {
        if (b.status === 'running') return fail(res, 409, 'benchmark_running', 'Another benchmark is already running');
      }
      const threads = Math.max(1, Math.min(256, Number(body.threads) || scenario.threads));
      const duration = Math.max(2, Math.min(600, Number(body.duration_seconds) || scenario.duration_seconds));
      const b = startBenchmark(scenario, threads, duration);
      return ok(res, { id: b.id, scenario: b.scenario, start_time: b.start_time });
    }

    // Used by the legacy /performance dashboard: starts the first scenario.
    if (m === 'POST' && p === '/api/performance/benchmarks') {
      for (const b of benchmarks.values()) {
        if (b.status === 'running') return fail(res, 409, 'benchmark_running', 'Another benchmark is already running');
      }
      const b = startBenchmark(SCENARIOS[0], SCENARIOS[0].threads, 20);
      return ok(res, { id: b.id, scenario: b.scenario, start_time: b.start_time });
    }

    if (m === 'GET' && p === '/api/performance/benchmarks/compare') {
      const a = benchmarks.get(u.searchParams.get('a'));
      const b = benchmarks.get(u.searchParams.get('b'));
      if (!a || !b) return fail(res, 404, 'benchmark_not_found', 'Benchmark not found');
      if (!a.summary || !b.summary) return fail(res, 409, 'benchmark_running', 'Benchmark has not finished yet');
      const sa = statsOf(a);
      const sb = statsOf(b);
      return ok(res, {
        a: sa,
        b: sb,
        delta: {
          qps_pct: pct(sa.qps, sb.qps),
          avg_latency_pct: pct(sa.avg_latency_ms, sb.avg_latency_ms),
          p95_pct: pct(sa.p95_latency_ms, sb.p95_latency_ms),
          p99_pct: pct(sa.p99_latency_ms, sb.p99_latency_ms),
        },
      });
    }

    if ((match = p.match(/^\/api\/performance\/benchmarks\/([^/]+)\/(stop|stream|samples)$/))) {
      const b = benchmarks.get(match[1]);
      if (!b) return fail(res, 404, 'benchmark_not_found', 'Benchmark not found');
      if (match[2] === 'stop' && m === 'POST') {
        if (b.status !== 'running') return fail(res, 409, 'not_running', 'Benchmark is not running');
        finishBenchmark(b, 'cancelled');
        return ok(res, { id: b.id, status: b.status });
      }
      if (match[2] === 'stream' && m === 'GET') return handleStream(req, res, b, u.searchParams);
      if (match[2] === 'samples' && m === 'GET') {
        const since = Number(u.searchParams.get('since')) || 0;
        return ok(res, { status: b.status, samples: b.samples.filter((s) => s.seq > since) });
      }
    }

    if (p.startsWith('/api/') || p.startsWith('/ws/')) return fail(res, 404, 'not_found', 'Endpoint not implemented in mock');

    // --- Pages + static ---
    if (m === 'GET') {
      if (p === '/' || p === '/benchmark-live') return serveFile(res, path.join(WEB_ROOT, 'templates', 'benchmark_live.html'));
      if (p === '/performance') return serveFile(res, path.join(WEB_ROOT, 'templates', 'performance_dashboard.html'));
      if (p === '/graph') return serveFile(res, path.join(WEB_ROOT, 'templates', 'visualization.html'));
      if (p.startsWith('/static/')) return serveFile(res, path.join(WEB_ROOT, p));
    }
    return fail(res, 404, 'not_found', 'Not found');
  } catch (err) {
    console.error(err);
    if (!res.headersSent) fail(res, 500, 'internal', String(err && err.message));
  }
});

server.listen(PORT, () => {
  console.log(`mock server on http://localhost:${PORT}/ (tick ${TICK_MS} ms)`);
});

function shutdown() {
  for (const b of benchmarks.values()) clearInterval(b.timer);
  server.close(() => process.exit(0));
  setTimeout(() => process.exit(0), 500).unref();
}
process.on('SIGINT', shutdown);
process.on('SIGTERM', shutdown);
