'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const { spawn } = require('node:child_process');
const { once } = require('node:events');
const { readFileSync } = require('node:fs');
const path = require('node:path');
const net = require('node:net');
const vm = require('node:vm');
const d3 = require('../static/js/vendor/d3.min.js');

async function mock(t, mode, database = 'mysql') {
  const process = spawn(global.process.execPath, [path.join(__dirname, 'mock-server.js')], {
    env: { ...global.process.env, PORT: '0', TICK_MS: '50', MODE: mode, DB_TYPE: database },
    stdio: ['ignore', 'pipe', 'pipe'],
  });
  t.after(async () => {
    if (process.exitCode === null) {
      process.kill('SIGTERM');
      await once(process, 'exit');
    }
  });
  const address = await new Promise((resolve, reject) => {
    let output = '';
    process.stdout.on('data', (chunk) => {
      output += chunk;
      const match = output.match(/localhost:(\d+)\//);
      if (match) resolve(`http://127.0.0.1:${match[1]}`);
    });
    process.on('error', reject);
    process.on('exit', (code) => reject(new Error(`Mock exited: ${code}`)));
  });
  return {
    address,
    async get(route) {
      const response = await fetch(address + route);
      const body = await response.json();
      return { status: response.status, ...body };
    },
    async post(route, data) {
      const response = await fetch(address + route, {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(data || {}),
      });
      return { status: response.status, ...await response.json() };
    },
  };
}

async function completed(api, id) {
  for (let i = 0; i < 100; i++) {
    const response = await api.get(`/api/performance/benchmarks/${id}/samples?since=0`);
    if (response.data.status !== 'running') return response.data;
    await new Promise((resolve) => setTimeout(resolve, 25));
  }
  throw new Error('Mock benchmark did not finish');
}

function subscribeRaw(address) {
  const url = new URL(address);
  return new Promise((resolve, reject) => {
    const socket = net.connect(Number(url.port), url.hostname);
    let buffer = Buffer.alloc(0), upgraded = false;
    const messages = [];
    socket.on('error', reject);
    socket.on('connect', () => {
      socket.write('GET /ws/performance HTTP/1.1\r\nHost: localhost\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n\r\n');
    });
    socket.on('data', (chunk) => {
      buffer = Buffer.concat([buffer, chunk]);
      if (!upgraded) {
        const end = buffer.indexOf('\r\n\r\n');
        if (end === -1) return;
        assert.match(buffer.subarray(0, end).toString(), /101 Switching Protocols/);
        assert.match(buffer.subarray(0, end).toString(), /s3pPLMBiTxaQ9kYGzzhZRbK\+xOo=/);
        buffer = buffer.subarray(end + 4);
        upgraded = true;
        const payload = Buffer.from(JSON.stringify({ type: 'subscribe', topic: 'benchmark' }));
        const mask = Buffer.from([1, 2, 3, 4]);
        const frame = Buffer.alloc(6 + payload.length);
        frame[0] = 0x81; frame[1] = 0x80 | payload.length; mask.copy(frame, 2);
        payload.forEach((value, index) => { frame[index + 6] = value ^ mask[index % 4]; });
        // Split the subscription to exercise buffered frame parsing.
        socket.write(frame.subarray(0, 3));
        socket.write(frame.subarray(3));
        resolve({ socket, messages });
      }
      while (buffer.length >= 2) {
        let length = buffer[1] & 127, offset = 2;
        if (length === 126) { if (buffer.length < 4) return; length = buffer.readUInt16BE(2); offset = 4; }
        if (length === 127) { if (buffer.length < 10) return; length = Number(buffer.readBigUInt64BE(2)); offset = 10; }
        if (buffer.length < offset + length) return;
        messages.push(JSON.parse(buffer.subarray(offset, offset + length).toString()));
        buffer = buffer.subarray(offset + length);
      }
    });
  });
}

for (const mode of ['demo', 'generic']) {
  test(`${mode}: config, stream, history, suggestions, compare and WebSocket`, { timeout: 15000 }, async (t) => {
    const api = await mock(t, mode, mode === 'generic' ? 'postgresql' : 'mysql');
    const config = (await api.get('/api/performance/live/config')).data;
    assert.equal(config.mode, mode);
    assert.equal(config.database_type, mode === 'generic' ? 'postgresql' : 'mysql');
    assert.equal(config.optimization.available, mode === 'demo');
    assert.equal(config.scenarios.length > 0, mode === 'demo');
    assert.equal((await api.get('/api/performance/benchmarks/history')).data.runs.length, 0);
    const topology = (await api.get('/api/performance/topology')).data;
    for (const edge of topology.edges) assert.equal(edge.id, [edge.source, edge.target].sort().join('-'));
    const mapping = (await api.get('/api/performance/live/mapping')).data;
    assert.equal(mapping.labels.Order.table, 'orders');
    assert.deepEqual(mapping.relationships.CONTAINS_PRODUCT.edges, ['order_items-orders', 'order_items-products']);
    const graph = await api.get('/api/graph');
    assert.equal(graph.nodes.length, 4);
    assert.equal(graph.relationships.length, 3);
    const client = await subscribeRaw(api.address);
    t.after(() => client.socket.destroy());
    const started = await api.post('/api/performance/benchmarks', {
      tool: 'custom', query_set: config.query_sets[0].name, threads: 4, duration_seconds: 3,
    });
    assert.equal(started.status, 200);
    const id = started.data.id;
    assert.equal((await api.get(`/api/performance/optimization/suggestions?benchmark_id=${id}`)).status, 409);
    const replay = await completed(api, id);
    assert.equal(replay.samples.length, 3);
    assert.equal(replay.samples[0].threads, 4);
    assert.ok(client.messages.some((message) => message.topic === 'benchmark' && message.data.benchmark_id === id));
    const stream = await fetch(`${api.address}/api/performance/benchmarks/${id}/stream`).then((response) => response.text());
    assert.match(stream, /event: sample/);
    assert.match(stream, /event: done/);
    const suggestions = await api.get(`/api/performance/optimization/suggestions?benchmark_id=${id}`);
    assert.equal(suggestions.data.suggestions.length, 2);
    assert.ok(suggestions.data.suggestions[0].statements[0].startsWith('CREATE INDEX'));
    assert.equal((await api.get('/api/performance/optimization/suggestions?benchmark_id=unknown')).status, 404);
    let second;
    if (mode === 'demo') {
      assert.equal((await api.post('/api/demo/optimization/apply')).data.applied, true);
      second = await api.post(`/api/demo/scenarios/${config.scenarios[0].name}/run`, { threads: 4, duration_seconds: 3 });
    } else {
      assert.equal((await api.post('/api/demo/optimization/apply')).status, 403);
      second = await api.post('/api/performance/benchmarks', { tool: 'custom', query_set: config.query_sets[0].name, threads: 4, duration_seconds: 3 });
    }
    await completed(api, second.data.id);
    const history = await api.get('/api/performance/benchmarks/history?limit=20');
    assert.equal(history.data.runs.length, 2);
    assert.equal(history.data.runs[0].id, second.data.id);
    assert.equal(history.data.runs[0].labels.index_applied, String(mode === 'demo'));
    const comparison = await api.get(`/api/performance/benchmarks/compare?a=${id}&b=${second.data.id}`);
    assert.equal(comparison.status, 200);
    if (mode === 'demo') {
      assert.ok(comparison.data.delta.qps_pct > 0);
      assert.equal((await api.get(`/api/performance/optimization/suggestions?benchmark_id=${second.data.id}`)).data.suggestions.length, 0);
      assert.equal((await api.post('/api/demo/optimization/revert')).data.applied, false);
    }
    const stopped = await api.post('/api/performance/benchmarks', { tool: 'custom', query_set: config.query_sets[0].name, threads: 2, duration_seconds: 60 });
    assert.equal((await api.post(`/api/performance/benchmarks/${stopped.data.id}/stop`)).data.status, 'cancelled');
    assert.equal((await api.get('/api/performance/benchmarks/history?query_set=unknown')).data.runs.length, 0);
  });
}

function loadScript(context, name) {
  vm.runInContext(readFileSync(path.join(__dirname, '../static/js', name), 'utf8'), context, { filename: name });
}

test('explicit overlay mapping, shared colour scale, no-data handling and restoration', async () => {
  const elements = { liveBenchToggle: { checked: true }, liveBenchStatus: { textContent: '' } };
  const context = vm.createContext({
    d3, window: { addEventListener() {} }, document: { getElementById: (id) => elements[id] },
    fetch: async () => ({ ok: false, status: 404 }),
  });
  loadScript(context, 'live-graph.js');
  loadScript(context, 'live-benchmark-overlay.js');
  const shared = context.window.LiveGraph;
  assert.equal(shared.logHeat(5), 0);
  assert.equal(shared.logHeat(500), 1);
  assert.equal(shared.heatColor(0), 'rgb(34, 197, 94)');
  const nodeValues = new Map([['order', { id: 'order', color: 'blue', size: 25 }], ['unmapped', { id: 'unmapped', color: 'purple', size: 10 }]]);
  const edgeValues = new Map([['edge', { id: 'edge', color: 'gray', width: 2 }]]);
  const store = (values) => ({ update: (updates) => updates.forEach((row) => values.set(row.id, { ...values.get(row.id), ...row })) });
  const visualizer = {
    currentGraphNodes: [{ id: 'order', label: 'Order' }, { id: 'unmapped', label: 'Unknown' }],
    currentGraphEdges: [{ id: 'edge', type: 'PLACED_ORDER' }],
    originalNodeVisuals: new Map([...nodeValues].map(([id, row]) => [id, { ...row }])),
    originalEdgeVisuals: new Map([...edgeValues].map(([id, row]) => [id, { ...row }])),
    network: { body: { data: { nodes: store(nodeValues), edges: store(edgeValues) } } },
  };
  const overlay = new context.window.LiveBenchmarkOverlay(visualizer);
  overlay.mapping = { labels: { Order: { table: 'orders' } }, relationships: { PLACED_ORDER: { edge: 'customers-orders' } } };
  overlay.apply({ benchmark_id: 'run-1', graph: { nodes: [{ id: 'orders', heat: 1, qps: 100 }], edges: [{ id: 'customers-orders', avg_latency_ms: 500, load: 1 }] } });
  assert.equal(nodeValues.get('order').color.background, shared.heatColor(1));
  assert.equal(nodeValues.get('order').size, 50);
  assert.equal(nodeValues.get('unmapped').color, 'purple');
  assert.equal(edgeValues.get('edge').color.color, shared.heatColor(1));
  assert.equal(edgeValues.get('edge').width, 14);
  overlay.mapping.relationships.PLACED_ORDER.edges = ['customers-orders', 'join-edge', 'missing-edge'];
  overlay.apply({ benchmark_id: 'run-1', graph: { nodes: [], edges: [
    { id: 'customers-orders', avg_latency_ms: 5, load: 0.8 },
    { id: 'join-edge', avg_latency_ms: 50, load: 0.2 },
  ] } });
  assert.equal(edgeValues.get('edge').color.color, shared.heatColor(shared.logHeat(50)));
  assert.equal(edgeValues.get('edge').width, 3 + 11 * Math.pow(0.8, 0.9));
  overlay.apply({ benchmark_id: 'sysbench', graph: { nodes: [], edges: [] } });
  assert.equal(nodeValues.get('order').color, 'blue');
  assert.equal(edgeValues.get('edge').width, 2);
  overlay.disable();
  assert.equal(nodeValues.get('order').size, 25);
  assert.equal(elements.liveBenchToggle.checked, false);
  elements.liveBenchToggle.checked = true;
  await overlay.enable();
  assert.equal(elements.liveBenchToggle.checked, false);
  assert.equal(elements.liveBenchStatus.textContent, 'No benchmark data yet - run one on /benchmark-live');
});

function fakeElement(id) {
  const listeners = {};
  let html = '';
  return {
    id, dataset: {}, style: {}, value: '', hidden: false, disabled: false, textContent: '',
    classList: { toggle() {}, contains() { return false; } },
    addEventListener(event, callback) { listeners[event] = callback; },
    dispatch(event, target) { return listeners[event] && listeners[event]({ currentTarget: this, target: target || this }); },
    blur() {},
    get innerHTML() { return html; },
    set innerHTML(value) {
      html = value;
      if (id === 'scenarioSelect') this.value = (value.match(/value="([^"]*)"/) || [])[1] || '';
    },
  };
}

async function flush() {
  await new Promise((resolve) => setImmediate(resolve));
  await new Promise((resolve) => setImmediate(resolve));
}

function pageHarness(mode, legacy = false) {
  const html = readFileSync(path.join(__dirname, '../templates/benchmark_live.html'), 'utf8');
  const elements = Object.fromEntries([...html.matchAll(/id="([^"]+)"/g)].map((match) => [match[1], fakeElement(match[1])]));
  const documentListeners = {}, windowListeners = {}, requests = [], timers = [], streams = [], sockets = [];
  const graphCalls = [];
  const row = (id, started) => ({
    id, scenario: 'workload', query_set: 'workload', status: 'completed', started_at: started,
    duration_seconds: 3, qps: 42, avg_latency_ms: 5, p95_latency_ms: 9, p99_latency_ms: 12,
    total_queries: 126, has_samples: true, labels: { index_applied: id === 'new' ? 'true' : 'false' },
  });
  const rows = [row('new', '2026-01-02T00:00:00Z'), row('old', '2026-01-01T00:00:00Z')];
  const topology = { nodes: [{ id: 'orders' }], edges: [] };
  const scenarios = [{ name: 'workload', title: 'Workload', threads: 4, duration_seconds: 3 }];
  const optimization = { available: mode === 'demo', can_apply: mode === 'demo', applied: false, statements: [] };
  const config = { mode, database_type: 'postgresql', query_sets: scenarios, scenarios, topology, optimization };
  const sample = (seq, id = 'owned') => ({
    benchmark_id: id, seq, threads: 4, elapsed_seconds: seq,
    interval: { queries: 42, qps: 42, avg_latency_ms: 5, p50_latency_ms: 4, p95_latency_ms: 9, p99_latency_ms: 12, max_latency_ms: 13, errors: 0 },
    graph: { nodes: [{ id: 'orders', qps: 42, heat: 0 }], edges: [] }, queries: [],
  });
  const context = vm.createContext({
    d3, console, location: { protocol: 'http:', host: 'localhost' },
    getComputedStyle: () => ({ getPropertyValue: () => '#999999' }),
    localStorage: { getItem: () => 'dark', setItem() {} },
    setTimeout(callback, milliseconds) { const timer = { callback, milliseconds }; timers.push(timer); return timer; },
    clearTimeout(timer) { if (timer) timer.cancelled = true; },
    document: {
      body: fakeElement('body'), documentElement: {}, getElementById: (id) => elements[id],
      querySelector: () => fakeElement('index-actions'), querySelectorAll: () => [],
      addEventListener(event, callback) { documentListeners[event] = callback; },
    },
    window: { addEventListener(event, callback) { windowListeners[event] = callback; },
      LiveGraph: { create: () => Object.fromEntries(['init', 'update', 'reset', 'recolor', 'destroy'].map((method) =>
        [method, (data) => graphCalls.push({ method, data })])) },
    },
    Chart: class {
      constructor(element, config) { this.data = config.data; this.options = config.options; }
      update() {}
      resize() {}
    },
    EventSource: class {
      constructor(url) { this.url = url; this.listeners = {}; streams.push(this); }
      addEventListener(event, callback) { this.listeners[event] = callback; }
      emit(event, data) { this.listeners[event]({ data: JSON.stringify(data) }); }
      close() { this.closed = true; }
    },
    WebSocket: class {
      constructor(url) { this.url = url; sockets.push(this); }
      send(message) { this.sent = JSON.parse(message); }
      close() { this.closed = true; }
    },
    fetch: async (url, options = {}) => {
      requests.push({ url, method: options.method || 'GET', body: options.body && JSON.parse(options.body) });
      let data = {}, status = 200;
      if (url === '/api/performance/live/config') { data = config; if (legacy) status = 404; }
      else if (url === '/api/demo/scenarios') data = scenarios;
      else if (url === '/api/demo/topology') data = topology;
      else if (url === '/api/demo/optimization') data = optimization;
      else if (url === '/api/performance/benchmarks' || /\/run$/.test(url)) data = { id: 'owned' };
      else if (url.includes('/history')) data = { runs: rows };
      else if (url.includes('/samples')) data = { status: 'completed', samples: [sample(1, 'old'), sample(2, 'old'), sample(3, 'old')] };
      else if (url.includes('/compare?')) data = { a: rows[1], b: rows[0], delta: { qps_pct: 10, p95_pct: -20 } };
      else if (url.includes('/suggestions?')) data = { suggestions: [{ title: '<unsafe>', table: 'orders', columns: ['customer_id'], statements: ['CREATE INDEX x ON orders (customer_id)'], rollback: ['DROP INDEX x'], confidence: 0.8, source: 'explain' }] };
      return { ok: status === 200, status, json: async () => ({ success: status === 200, data }) };
    },
  });
  context.window.WebSocket = context.WebSocket;
  loadScript(context, 'benchmark-live.js');
  return { context, elements, documentListeners, windowListeners, requests, timers, streams, sockets, graphCalls, sample };
}

for (const [mode, legacy] of [['generic', false], ['demo', false], ['demo', true]]) {
  test(`benchmark page logic: ${mode}${legacy ? ' legacy fallback' : ''}`, async () => {
    const page = pageHarness(mode, legacy);
    await page.documentListeners.DOMContentLoaded();
    await flush();
    const { elements, streams, requests } = page;
    assert.match(elements.modeDescription.textContent, mode === 'demo' ? /Demo mode/ : /Generic mode/);
    assert.equal(elements.indexPanel.hidden, mode !== 'demo');
    assert.equal(elements.startBtn.disabled, false);
    assert.equal(elements.threadsInput.value, 4);
    assert.ok(page.graphCalls.some((call) => call.method === 'init'));
    elements.startBtn.dispatch('click');
    await flush();
    const request = requests.find((request) => request.method === 'POST');
    assert.equal(request.url, mode === 'demo' ? '/api/demo/scenarios/workload/run' : '/api/performance/benchmarks');
    assert.equal(request.body.tool, 'custom');
    assert.equal(request.body.query_set, 'workload');
    assert.equal(elements.startBtn.disabled, true);
    streams[0].emit('sample', page.sample(1));
    streams[0].emit('sample', page.sample(2));
    streams[0].emit('sample', page.sample(3));
    assert.equal(elements.kQps.textContent, '42.0');
    streams[0].emit('done', { status: 'completed' });
    await flush();
    assert.equal(elements.startBtn.disabled, false);
    assert.match(elements.suggestionsBody.innerHTML, /&lt;unsafe&gt;/);
    assert.match(elements.suggestionsBody.innerHTML, /CREATE INDEX/);
    assert.match(elements.historyBody.innerHTML, /index applied/);
    elements.historyBody.dispatch('change', { dataset: { runSelect: 'new' }, checked: true });
    elements.historyBody.dispatch('change', { dataset: { runSelect: 'old' }, checked: true });
    elements.historyCompare.dispatch('click');
    await flush();
    assert.ok(requests.some((request) => request.url.endsWith('/compare?a=old&b=new')));
    assert.match(elements.compareBody.innerHTML, /Run B/);
    elements.historyBody.dispatch('click', { closest: () => ({ dataset: { runOpen: 'old' } }) });
    await flush();
    assert.equal(elements.connPill.textContent, 'Historical run');
    assert.equal(elements.kQps.textContent, '42.0');
    const socket = page.sockets[0];
    socket.onopen();
    assert.equal(socket.sent.topic, 'benchmark');
    socket.onmessage({ data: JSON.stringify({ type: 'data', topic: 'benchmark', data: page.sample(1, 'external') }) });
    assert.equal(elements.connPill.textContent, 'External run (live)');
    assert.equal(elements.stopBtn.disabled, true);
    const inactivity = page.timers.findLast((timer) => timer.milliseconds === 5000 && !timer.cancelled);
    assert.ok(inactivity);
    await inactivity.callback();
    await flush();
    assert.equal(elements.connPill.textContent, 'Completed');
    page.windowListeners.pagehide();
    assert.equal(socket.closed, true);
    assert.ok(page.graphCalls.some((call) => call.method === 'destroy'));
  });
}
