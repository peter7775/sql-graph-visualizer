/*
 * Live benchmark page: runs a demo scenario against the API, streams samples
 * over SSE (polling fallback) and renders them as a heating table graph (D3),
 * rolling charts (Chart.js), a slowest-query list and a before/after
 * comparison.
 */
(function () {
    'use strict';

    // ------------------------------------------------------------------
    // Constants and helpers
    // ------------------------------------------------------------------

    const API = {
        scenarios: '/api/demo/scenarios',
        topology: '/api/demo/topology',
        optimization: '/api/demo/optimization',
        run: (name) => `/api/demo/scenarios/${encodeURIComponent(name)}/run`,
        stop: (id) => `/api/performance/benchmarks/${encodeURIComponent(id)}/stop`,
        stream: (id) => `/api/performance/benchmarks/${encodeURIComponent(id)}/stream`,
        samples: (id, since) => `/api/performance/benchmarks/${encodeURIComponent(id)}/samples?since=${since}`,
        compare: (a, b) => `/api/performance/benchmarks/compare?a=${encodeURIComponent(a)}&b=${encodeURIComponent(b)}`,
    };

    const CHART_WINDOW_SECONDS = 60;
    const MAX_QUERIES_SHOWN = 6;
    const MAX_PARTICLES = 12;

    const $ = (id) => document.getElementById(id);
    const clamp = (v, lo, hi) => Math.min(hi, Math.max(lo, v));

    const heatColor = d3.scaleLinear()
        .domain([0, 0.5, 1])
        .range(['#22c55e', '#eab308', '#ef4444'])
        .interpolate(d3.interpolateRgb)
        .clamp(true);

    // Latency (ms) -> heat 0..1, used where the server gives no heat value.
    const latencyHeat = d3.scaleLinear().domain([0, 50, 200, 500]).range([0, 0.3, 0.8, 1]).clamp(true);

    // Absolute latency heat on the same log scale the server uses for nodes
    // (5 ms = cool, 500 ms = hot). Edge colour uses this so a fast join stays
    // green even when it carries a large share of the load.
    function logHeat(ms) {
        if (!isFinite(ms) || ms <= 5) return 0;
        return clamp(Math.log(ms / 5) / Math.log(100), 0, 1);
    }

    function fmtNum(v) {
        if (!isFinite(v)) return '-';
        return v >= 100 ? Math.round(v).toLocaleString('en-US') : v.toFixed(1);
    }

    function fmtInt(v) {
        return isFinite(v) ? Math.round(v).toLocaleString('en-US') : '-';
    }

    function fmtMs(v) {
        if (!isFinite(v)) return '-';
        return v.toFixed(v >= 100 ? 0 : v >= 10 ? 1 : 2);
    }

    function esc(s) {
        return String(s == null ? '' : s).replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
    }

    function cssVar(name) {
        return getComputedStyle(document.body).getPropertyValue(name).trim();
    }

    function deltaInfo(pct, lowerIsBetter) {
        if (pct == null || !isFinite(pct)) return null;
        if (Math.abs(pct) < 0.5) return { text: '+/-0 %', cls: 'neutral' };
        const abs = Math.abs(pct);
        const num = abs >= 10 ? Math.round(abs) : abs.toFixed(1);
        const good = (pct < 0) === lowerIsBetter;
        return { text: `${pct > 0 ? '+' : '-'}${num} %`, cls: good ? 'good' : 'bad' };
    }

    class ApiError extends Error {}

    async function api(method, url, body) {
        let res;
        try {
            res = await fetch(url, {
                method,
                headers: body ? { 'Content-Type': 'application/json' } : undefined,
                body: body ? JSON.stringify(body) : undefined,
            });
        } catch (e) {
            throw new ApiError('Cannot reach the API server. Is the backend running?');
        }
        let json = null;
        try {
            json = await res.json();
        } catch (e) { /* non-JSON body */ }
        if (!res.ok || !json || json.success === false) {
            const msg = json && json.error && json.error.message ? json.error.message : `Request failed (HTTP ${res.status})`;
            throw new ApiError(msg);
        }
        return json.data;
    }

    // ------------------------------------------------------------------
    // State
    // ------------------------------------------------------------------

    const state = {
        scenarios: [],
        optimization: null,
        topologyReady: false,
        current: null,      // run in progress or last finished: {id, samples, ...}
        slots: { A: null, B: null },
        ghost: null,        // previous completed run shown as dashed line
        cmp: null,          // {a, b, delta, source}
        starting: false,
        stopping: false,
        indexBusy: false,
    };

    const isRunning = () => !!(state.current && !state.current.finished);

    // ------------------------------------------------------------------
    // Error banner and connection pill
    // ------------------------------------------------------------------

    function showError(msg) {
        $('errorText').textContent = msg;
        $('errorBanner').hidden = false;
    }

    function clearError() {
        $('errorBanner').hidden = true;
    }

    function setPill(stateName, text) {
        const pill = $('connPill');
        pill.dataset.state = stateName;
        pill.textContent = text;
    }

    // ------------------------------------------------------------------
    // Graph (D3)
    // ------------------------------------------------------------------

    const Graph = (function () {
        const W = 900;
        const H = 520;
        let svg, edgeLayer, particleLayer, nodeLayer;
        let nodes = [];
        let edges = [];
        let nodeById = new Map();
        let edgeById = new Map();
        let maxQps = 1;
        let hover = null;
        let rafId = null;
        let lastTs = 0;
        const reduceMotion = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;

        function init(topology) {
            svg = d3.select('#graphSvg');
            svg.selectAll('*').remove();
            svg.attr('viewBox', `0 0 ${W} ${H}`).attr('preserveAspectRatio', 'xMidYMid meet');

            nodes = (topology.nodes || []).map((n) => ({
                id: n.id, table: n.table || n.id, label: n.label || n.table || n.id,
                qps: 0, lat: 0, heat: 0, queries: 0, active: false,
            }));
            nodeById = new Map(nodes.map((n) => [n.id, n]));
            edges = (topology.edges || [])
                .filter((e) => nodeById.has(e.source) && nodeById.has(e.target))
                .map((e) => ({
                    id: e.id || `${e.source}-${e.target}`, relation: e.relation || '',
                    s: nodeById.get(e.source), t: nodeById.get(e.target),
                    qps: 0, lat: 0, load: 0, rank: '', active: false,
                    n: 0, speed: 0.2, phase: Math.random(), color: '#888', dots: [],
                }));
            edgeById = new Map(edges.map((e) => [e.id, e]));

            layout();
            // Put the label on the side away from the node's edges.
            nodes.forEach((n) => {
                const dy = edges.reduce((sum, e) => {
                    if (e.s === n) return sum + (e.t.y - n.y);
                    if (e.t === n) return sum + (e.s.y - n.y);
                    return sum;
                }, 0);
                n.labelUp = dy > 20;
            });

            edgeLayer = svg.append('g');
            particleLayer = svg.append('g').style('pointer-events', 'none');
            nodeLayer = svg.append('g');

            const edgeSel = edgeLayer.selectAll('g').data(edges, (d) => d.id).enter().append('g').attr('class', 'edge');
            edgeSel.append('line').attr('class', 'edge-hit')
                .attr('x1', (d) => d.s.x).attr('y1', (d) => d.s.y).attr('x2', (d) => d.t.x).attr('y2', (d) => d.t.y);
            edgeSel.append('line').attr('class', 'edge-line')
                .attr('x1', (d) => d.s.x).attr('y1', (d) => d.s.y).attr('x2', (d) => d.t.x).attr('y2', (d) => d.t.y)
                .style('pointer-events', 'none');
            edgeSel.append('text').attr('class', 'edge-label')
                .attr('x', (d) => (d.s.x + d.t.x) / 2).attr('y', (d) => (d.s.y + d.t.y) / 2 + 4);
            edgeSel.on('mouseenter mousemove', (ev, d) => showTip(ev, { type: 'edge', id: d.id }))
                .on('mouseleave', hideTip);

            edges.forEach((e) => {
                for (let i = 0; i < MAX_PARTICLES; i++) {
                    e.dots.push(particleLayer.append('circle').attr('r', 3).style('display', 'none').node());
                }
            });

            const nodeSel = nodeLayer.selectAll('g').data(nodes, (d) => d.id).enter().append('g')
                .attr('class', 'node').attr('transform', (d) => `translate(${d.x},${d.y})`);
            nodeSel.append('circle').attr('class', 'node-circle').attr('r', 22).attr('stroke-width', 2.5);
            nodeSel.append('text').attr('class', 'node-label').text((d) => d.label);
            nodeSel.append('text').attr('class', 'node-sub');
            nodeSel.on('mouseenter mousemove', (ev, d) => showTip(ev, { type: 'node', id: d.id }))
                .on('mouseleave', hideTip);

            render(false);
            if (!rafId && !reduceMotion) rafId = requestAnimationFrame(frame);
        }

        // Force layout computed once, then frozen so the graph never jumps on updates.
        // Several deterministic seeds are tried and the layout with the fewest edge
        // crossings / node-edge overlaps wins, so labels stay readable.
        function layout() {
            const box = { x0: 120, x1: W - 120, y0: 110, y1: H - 95 };
            const incident = edges.map((e) => [nodes.indexOf(e.s), nodes.indexOf(e.t)]);
            const attempts = nodes.length > 40 ? 1 : 24;
            let best = null;
            for (let seed = 0; seed < attempts; seed++) {
                const rnd = d3.randomLcg(seed + 1);
                const simNodes = nodes.map((n) => ({ id: n.id, x: W / 2 + (rnd() - 0.5) * 300, y: H / 2 + (rnd() - 0.5) * 200 }));
                const links = edges.map((e) => ({ source: e.s.id, target: e.t.id }));
                const sim = d3.forceSimulation(simNodes)
                    .force('link', d3.forceLink(links).id((d) => d.id).distance(200).strength(0.9))
                    .force('charge', d3.forceManyBody().strength(-1600))
                    .force('x', d3.forceX(W / 2).strength(0.04))
                    .force('y', d3.forceY(H / 2).strength(0.14))
                    .force('collide', d3.forceCollide(80))
                    .stop();
                for (let i = 0; i < 400; i++) sim.tick();
                const pos = fitToBox(simNodes, box);
                const score = layoutScore(pos, incident);
                if (!best || score < best.score) best = { pos, score };
                if (score === 0) break;
            }
            best.pos.forEach((p, i) => { nodes[i].x = p.x; nodes[i].y = p.y; });
        }

        function fitToBox(simNodes, box) {
            const xs = simNodes.map((n) => n.x);
            const ys = simNodes.map((n) => n.y);
            const minX = Math.min(...xs), maxX = Math.max(...xs);
            const minY = Math.min(...ys), maxY = Math.max(...ys);
            const fit = (v, lo, hi, a, b) => (hi - lo < 1 ? (a + b) / 2 : a + ((v - lo) / (hi - lo)) * (b - a));
            return simNodes.map((s) => ({ x: fit(s.x, minX, maxX, box.x0, box.x1), y: fit(s.y, minY, maxY, box.y0, box.y1) }));
        }

        function layoutScore(pos, links) {
            const orient = (a, b, c) => (b.x - a.x) * (c.y - a.y) - (b.y - a.y) * (c.x - a.x);
            const crosses = (a, b, c, d) => orient(a, b, c) * orient(a, b, d) < 0 && orient(c, d, a) * orient(c, d, b) < 0;
            const dist = (p, a, b) => {
                const dx = b.x - a.x, dy = b.y - a.y;
                const t = clamp(((p.x - a.x) * dx + (p.y - a.y) * dy) / (dx * dx + dy * dy || 1), 0, 1);
                return Math.hypot(p.x - (a.x + t * dx), p.y - (a.y + t * dy));
            };
            let score = 0;
            for (let i = 0; i < links.length; i++) {
                for (let j = i + 1; j < links.length; j++) {
                    const [a, b] = links[i];
                    const [c, d] = links[j];
                    if (a === c || a === d || b === c || b === d) continue;
                    if (crosses(pos[a], pos[b], pos[c], pos[d])) score += 10000;
                }
                pos.forEach((p, k) => {
                    if (k === links[i][0] || k === links[i][1]) return;
                    const d = dist(p, pos[links[i][0]], pos[links[i][1]]);
                    if (d < 90) score += (90 - d) * (90 - d);
                });
            }
            return score;
        }

        function radius(n) {
            return 20 + 30 * Math.sqrt(clamp(n.qps / maxQps, 0, 1));
        }

        function render(animate) {
            const dur = animate ? 700 : 0;
            nodeLayer.selectAll('g.node').each(function (d) {
                const g = d3.select(this);
                const r = radius(d);
                const col = d.active ? heatColor(d.heat) : cssVar('--neutral-node');
                g.select('circle')
                    .transition('u').duration(dur).ease(d3.easeCubicOut)
                    .attr('r', r)
                    .style('fill', col)
                    .style('stroke', d3.color(col).darker(0.7).formatRgb());
                g.select('circle').style('filter', d.active && d.heat > 0.4
                    ? `drop-shadow(0 0 ${Math.round(4 + 14 * d.heat)}px ${col})` : 'none');
                g.select('.node-label').transition('u').duration(dur).attr('y', d.labelUp ? -(r + 27) : r + 19);
                g.select('.node-sub').transition('u').duration(dur).attr('y', d.labelUp ? -(r + 11) : r + 35);
                g.select('.node-sub').text(d.active ? `${fmtNum(d.qps)} qps - ${fmtMs(d.lat)} ms` : '');
            });

            edgeLayer.selectAll('g.edge').each(function (d) {
                const g = d3.select(this);
                const col = d.active ? heatColor(logHeat(d.lat)) : cssVar('--neutral-node');
                const w = d.active ? 3 + 11 * Math.pow(clamp(d.load, 0, 1), 0.9) : 2.5;
                d.color = col;
                g.select('.edge-line').transition('u').duration(dur)
                    .style('stroke', col).style('stroke-width', w + 'px');
                g.select('.edge-label').text(d.active && d.qps > 0 ? `${fmtMs(d.lat)} ms` : '');
                d.dots.forEach((el) => {
                    el.style.fill = d3.color(col).brighter(0.9).formatRgb();
                    el.setAttribute('r', (2.4 + w / 5).toFixed(1));
                });
            });
            updateParticles();
            if (hover) refreshTip();
        }

        function updateParticles() {
            edges.forEach((e) => {
                if (!e.active || e.qps <= 0) {
                    e.n = 0;
                } else {
                    e.n = Math.max(1, Math.round(1 + (MAX_PARTICLES - 1) * Math.sqrt(clamp(e.qps / maxQps, 0, 1))));
                }
                e.speed = 0.1 + 0.5 * (1 - clamp(e.load, 0, 1));
                e.dots.forEach((el, i) => {
                    el.style.display = i < e.n && !reduceMotion ? '' : 'none';
                });
            });
        }

        function frame(ts) {
            const dt = lastTs ? Math.min(0.1, (ts - lastTs) / 1000) : 0;
            lastTs = ts;
            edges.forEach((e) => {
                if (!e.n) return;
                e.phase = (e.phase + e.speed * dt) % 1;
                for (let i = 0; i < e.n; i++) {
                    const t = (e.phase + i / e.n) % 1;
                    e.dots[i].setAttribute('cx', (e.s.x + (e.t.x - e.s.x) * t).toFixed(1));
                    e.dots[i].setAttribute('cy', (e.s.y + (e.t.y - e.s.y) * t).toFixed(1));
                }
            });
            rafId = requestAnimationFrame(frame);
        }

        function update(graph) {
            if (!graph || !nodes.length) return;
            (graph.nodes || []).forEach((n) => {
                const node = nodeById.get(n.id);
                if (!node) return;
                node.active = true;
                node.qps = n.qps || 0;
                node.lat = n.avg_latency_ms || 0;
                node.heat = clamp(n.heat != null ? n.heat : latencyHeat(node.lat), 0, 1);
                node.queries = n.queries || 0;
                maxQps = Math.max(maxQps, node.qps);
            });
            (graph.edges || []).forEach((e) => {
                const edge = edgeById.get(e.id);
                if (!edge) return;
                edge.active = true;
                edge.qps = e.qps || 0;
                edge.lat = e.avg_latency_ms || 0;
                edge.load = clamp(e.load != null ? e.load : latencyHeat(edge.lat), 0, 1);
                edge.rank = e.rank || '';
                maxQps = Math.max(maxQps, edge.qps);
            });
            render(true);
        }

        function reset() {
            if (!nodes.length) return;
            nodes.forEach((n) => { n.active = false; n.qps = 0; n.lat = 0; n.heat = 0; });
            edges.forEach((e) => { e.active = false; e.qps = 0; e.lat = 0; e.load = 0; e.rank = ''; });
            render(true);
        }

        // --- tooltip ---
        function tipHtml(h) {
            if (h.type === 'node') {
                const n = nodeById.get(h.id);
                if (!n) return '';
                if (!n.active) return `<b>${esc(n.label)}</b><div class="t-muted">${esc(n.table)} - no data yet</div>`;
                return `<b>${esc(n.label)}</b> <span class="t-muted">${esc(n.table)}</span>`
                    + `<div class="t-row"><span>QPS</span><span>${fmtNum(n.qps)}</span></div>`
                    + `<div class="t-row"><span>Avg latency</span><span>${fmtMs(n.lat)} ms</span></div>`
                    + `<div class="t-row"><span>Queries</span><span>${fmtInt(n.queries)}</span></div>`
                    + `<div class="t-row"><span>Heat</span><span>${Math.round(n.heat * 100)} %</span></div>`;
            }
            const e = edgeById.get(h.id);
            if (!e) return '';
            const head = `<b>${esc(e.s.label)} - ${esc(e.t.label)}</b>`
                + (e.relation ? `<div class="t-muted">${esc(e.relation)}</div>` : '');
            if (!e.active) return head + '<div class="t-muted">no data yet</div>';
            return head
                + `<div class="t-row"><span>QPS</span><span>${fmtNum(e.qps)}</span></div>`
                + `<div class="t-row"><span>Avg latency</span><span>${fmtMs(e.lat)} ms</span></div>`
                + `<div class="t-row"><span>Load</span><span>${Math.round(e.load * 100)} %</span></div>`
                + (e.rank ? `<div class="t-row"><span>Rank</span><span>${esc(e.rank)}</span></div>` : '');
        }

        function showTip(ev, h) {
            hover = h;
            const tip = $('tooltip');
            tip.innerHTML = tipHtml(h);
            tip.hidden = false;
            const wrap = $('graphWrap');
            const [x, y] = d3.pointer(ev, wrap);
            const tw = tip.offsetWidth;
            const th = tip.offsetHeight;
            let left = x + 16;
            let top = y + 16;
            if (left + tw > wrap.clientWidth - 4) left = x - tw - 16;
            if (top + th > wrap.clientHeight - 4) top = y - th - 16;
            tip.style.left = Math.max(4, left) + 'px';
            tip.style.top = Math.max(4, top) + 'px';
        }

        function refreshTip() {
            const tip = $('tooltip');
            if (!tip.hidden) tip.innerHTML = tipHtml(hover);
        }

        function hideTip() {
            hover = null;
            $('tooltip').hidden = true;
        }

        return { init, update, reset, recolor: () => nodes.length && render(false) };
    })();

    // ------------------------------------------------------------------
    // Charts (Chart.js)
    // ------------------------------------------------------------------

    const Charts = (function () {
        let qps, lat;

        function baseOptions() {
            const text = cssVar('--muted');
            const grid = cssVar('--grid');
            return {
                responsive: true,
                maintainAspectRatio: false,
                animation: false,
                parsing: false,
                normalized: true,
                interaction: { mode: 'nearest', intersect: false, axis: 'x' },
                plugins: {
                    legend: { position: 'bottom', labels: { color: text, boxWidth: 12, boxHeight: 3, padding: 12 } },
                    tooltip: { callbacks: { title: (items) => (items.length ? `${items[0].parsed.x.toFixed(1)}s` : '') } },
                },
                elements: { point: { radius: 0, hoverRadius: 3 }, line: { tension: 0.3, borderWidth: 2 } },
                scales: {
                    x: { type: 'linear', min: 0, max: 30, ticks: { color: text, maxTicksLimit: 8, callback: (v) => `${Math.round(v)}s` }, grid: { color: grid } },
                    y: { beginAtZero: true, ticks: { color: text, maxTicksLimit: 6 }, grid: { color: grid } },
                },
            };
        }

        function ds(label, color, extra) {
            return Object.assign({ label, data: [], borderColor: color, backgroundColor: color, fill: false, spanGaps: true }, extra || {});
        }

        function init() {
            const ghost = { borderDash: [6, 4], borderColor: '#8b98a9', backgroundColor: '#8b98a9', borderWidth: 1.5, order: 10 };
            qps = new Chart($('qpsChart'), {
                type: 'line',
                data: { datasets: [
                    ds('QPS', '#4f9cf9', { backgroundColor: 'rgba(79,156,249,0.15)', fill: true }),
                    ds('Previous run', ghost.borderColor, ghost),
                ] },
                options: baseOptions(),
            });
            lat = new Chart($('latChart'), {
                type: 'line',
                data: { datasets: [
                    ds('p50', '#22c55e'),
                    ds('p95', '#eab308'),
                    ds('p99', '#ef4444'),
                    ds('Previous run p95', ghost.borderColor, ghost),
                ] },
                options: baseOptions(),
            });
        }

        function window_(duration, last) {
            const win = clamp(duration || 30, 10, CHART_WINDOW_SECONDS);
            // Whole seconds keep the axis bounds and tick labels tidy.
            const xmax = Math.max(Math.ceil(last), win);
            return { min: Math.max(0, xmax - win), max: xmax };
        }

        function ghostLabel() {
            const g = state.ghost;
            if (!g) return 'Previous run';
            const letter = state.slots.A === g ? 'Run A' : state.slots.B === g ? 'Run B' : 'Previous run';
            return `${letter} (${g.indexed ? 'index' : 'no index'})`;
        }

        function render() {
            const cur = state.current;
            const samples = cur ? cur.samples : [];
            const last = samples.length ? samples[samples.length - 1].elapsed_seconds : 0;
            const w = window_(cur ? cur.duration : 30, last);
            const pt = (s, f) => ({ x: s.elapsed_seconds, y: f(s) });
            // Latency is undefined for an interval without finished queries: leave a gap.
            const lpt = (s, f) => ({ x: s.elapsed_seconds, y: s.interval.queries > 0 ? f(s) : null });
            const inWin = (s) => s.elapsed_seconds >= w.min - 1;
            const vis = samples.filter(inWin);

            qps.data.datasets[0].data = vis.map((s) => pt(s, (x) => x.interval.qps));
            lat.data.datasets[0].data = vis.map((s) => lpt(s, (x) => x.interval.p50_latency_ms));
            lat.data.datasets[1].data = vis.map((s) => lpt(s, (x) => x.interval.p95_latency_ms));
            lat.data.datasets[2].data = vis.map((s) => lpt(s, (x) => x.interval.p99_latency_ms));

            const g = state.ghost;
            const gs = g ? g.samples.filter(inWin) : [];
            qps.data.datasets[1].data = gs.map((s) => pt(s, (x) => x.interval.qps));
            lat.data.datasets[3].data = gs.map((s) => lpt(s, (x) => x.interval.p95_latency_ms));
            qps.data.datasets[1].label = `${ghostLabel()} QPS`;
            lat.data.datasets[3].label = `${ghostLabel()} p95`;
            [qps, lat].forEach((c) => {
                c.options.scales.x.min = w.min;
                c.options.scales.x.max = w.max;
                c.update('none');
            });
        }

        function retheme() {
            [qps, lat].forEach((c) => {
                const o = baseOptions();
                c.options.plugins.legend.labels.color = o.plugins.legend.labels.color;
                c.options.scales.x.ticks.color = c.options.scales.y.ticks.color = cssVar('--muted');
                c.options.scales.x.grid.color = c.options.scales.y.grid.color = cssVar('--grid');
                c.update('none');
            });
        }

        return { init, render, retheme, resize: () => { qps.resize(); lat.resize(); } };
    })();

    // ------------------------------------------------------------------
    // KPI cards, query list, comparison
    // ------------------------------------------------------------------

    function renderKpis(s) {
        const cur = state.current;
        const iv = s.interval;
        $('kQps').textContent = fmtNum(iv.qps);
        $('kQpsSub').textContent = `avg ${fmtNum(cur.totalQueries / Math.max(1, s.elapsed_seconds))} qps - ${s.threads || cur.threads} threads`;
        // An interval in which no query finished (all workers stuck in slow
        // queries) has no latency data; keep the previous values instead of
        // showing a misleading 0 ms.
        if (iv.queries > 0) {
            setLatencyKpi('kP95', iv.p95_latency_ms);
            setLatencyKpi('kP99', iv.p99_latency_ms);
            $('kP95Sub').textContent = `p50 ${fmtMs(iv.p50_latency_ms)} ms - avg ${fmtMs(iv.avg_latency_ms)} ms`;
            $('kP99Sub').textContent = `max ${fmtMs(iv.max_latency_ms)} ms`;
        }
        const rate = cur.totalQueries ? (cur.totalErrors / cur.totalQueries) * 100 : 0;
        const errEl = $('kErr');
        errEl.textContent = `${rate >= 10 ? rate.toFixed(1) : rate.toFixed(2)} %`;
        errEl.style.color = cur.totalErrors > 0 ? cssVar('--bad') : '';
        $('kErrSub').textContent = `${fmtInt(cur.totalErrors)} errors / ${fmtInt(cur.totalQueries)} queries`;
    }

    function setLatencyKpi(id, ms) {
        const el = $(id);
        el.innerHTML = `${esc(fmtMs(ms))}<small>ms</small>`;
        el.style.color = heatColor(latencyHeat(ms));
    }

    function resetKpis() {
        ['kQps', 'kP95', 'kP99', 'kErr'].forEach((id) => { $(id).textContent = '-'; $(id).style.color = ''; });
        ['kQpsSub', 'kP95Sub', 'kP99Sub', 'kErrSub'].forEach((id) => { $(id).innerHTML = '&nbsp;'; });
        hideDeltas();
    }

    function hideDeltas() {
        ['dQps', 'dP95', 'dP99'].forEach((id) => { $(id).hidden = true; });
    }

    function applyDelta(id, pct, lowerIsBetter) {
        const el = $(id);
        const info = deltaInfo(pct, lowerIsBetter);
        if (!info) { el.hidden = true; return; }
        el.hidden = false;
        el.className = `bl-delta ${info.cls}`;
        el.textContent = info.text;
        el.title = 'Change of Run B versus Run A';
    }

    function renderQueries(queries) {
        const list = $('queryList');
        if (!queries || !queries.length) {
            list.innerHTML = '<p class="bl-muted">No query data in this sample.</p>';
            return;
        }
        const top = queries.slice().sort((a, b) => b.avg_latency_ms - a.avg_latency_ms).slice(0, MAX_QUERIES_SHOWN);
        const maxLat = Math.max(...top.map((q) => q.avg_latency_ms), 1);
        list.innerHTML = top.map((q) => {
            const col = heatColor(latencyHeat(q.avg_latency_ms));
            const title = esc(q.pattern || q.description || '');
            const chips = (q.tables || []).map((t) => `<span class="q-chip">${esc(t)}</span>`).join('');
            return `<div class="q-row" title="${title}">`
                + `<div class="q-bar" style="width:${(q.avg_latency_ms / maxLat * 100).toFixed(0)}%;background:${col}"></div>`
                + `<div class="q-main"><span class="q-desc">${esc(q.description || q.pattern)}</span><span class="q-tables">${chips}</span></div>`
                + `<div class="q-nums"><span class="q-lat" style="color:${col}">${fmtMs(q.avg_latency_ms)} ms</span>`
                + `<small>p95 ${fmtMs(q.p95_latency_ms || 0)} - ${fmtNum(q.qps || 0)} qps</small></div>`
                + '</div>';
        }).join('');
    }

    function runLabel(run) {
        if (state.slots.A === run) return 'Run A';
        if (state.slots.B === run) return 'Run B';
        return 'Run';
    }

    function statsOf(run) {
        const s = run.summary;
        return {
            id: run.id, scenario: run.scenario,
            qps: s.qps, avg_latency_ms: s.avg_latency_ms, p95_latency_ms: s.p95_latency_ms,
            p99_latency_ms: s.p99_latency_ms, total_queries: s.total_queries,
        };
    }

    function localCompare(a, b) {
        const pct = (x, y) => (x ? ((y - x) / x) * 100 : 0);
        const sa = statsOf(a);
        const sb = statsOf(b);
        return {
            a: sa, b: sb, source: 'local',
            delta: {
                qps_pct: pct(sa.qps, sb.qps),
                avg_latency_pct: pct(sa.avg_latency_ms, sb.avg_latency_ms),
                p95_pct: pct(sa.p95_latency_ms, sb.p95_latency_ms),
                p99_pct: pct(sa.p99_latency_ms, sb.p99_latency_ms),
            },
        };
    }

    function renderCompare() {
        const A = state.slots.A;
        const B = state.slots.B;
        const body = $('compareBody');
        const hint = $('compareHint');
        if (!A) {
            hint.textContent = '';
            body.innerHTML = '<p class="bl-muted">Run the scenario to record Run A.</p>';
            return;
        }
        const tag = (r) => `<span class="cmp-sub">${r.indexed ? 'index applied' : 'no index'}</span>`;
        const rows = [
            ['QPS', 'qps', false, fmtNum],
            ['Avg latency', 'avg_latency_ms', true, (v) => `${fmtMs(v)} ms`],
            ['p95 latency', 'p95_latency_ms', true, (v) => `${fmtMs(v)} ms`],
            ['p99 latency', 'p99_latency_ms', true, (v) => `${fmtMs(v)} ms`],
            ['Total queries', 'total_queries', false, fmtInt],
        ];
        const deltaKey = { qps: 'qps_pct', avg_latency_ms: 'avg_latency_pct', p95_latency_ms: 'p95_pct', p99_latency_ms: 'p99_pct' };

        if (!B) {
            const a = statsOf(A);
            hint.textContent = A.scenario || '';
            body.innerHTML = '<table class="cmp-table"><thead><tr><th>Metric</th><th>Run A' + tag(A) + '</th></tr></thead><tbody>'
                + rows.map((r) => `<tr><td>${r[0]}</td><td>${r[3](a[r[1]])}</td></tr>`).join('')
                + '</tbody></table>'
                + '<p class="bl-muted">Apply the suggested index and run again to see the difference.</p>';
            return;
        }

        const cmp = state.cmp || localCompare(A, B);
        hint.textContent = cmp.source === 'local' ? 'computed locally' : (B.scenario || '');
        const d = cmp.delta || {};
        const headline = [
            ['p95', d.p95_pct, true], ['p99', d.p99_pct, true], ['QPS', d.qps_pct, false], ['avg', d.avg_latency_pct, true],
        ].map(([name, pct, lower]) => {
            const info = deltaInfo(pct, lower);
            return info ? `<span class="bl-delta ${info.cls}">${name} ${info.text}</span>` : '';
        }).join('');
        body.innerHTML = `<div class="cmp-headline">${headline}</div>`
            + '<table class="cmp-table"><thead><tr><th>Metric</th><th>Run A' + tag(A) + '</th><th>Run B' + tag(B) + '</th><th>Change</th></tr></thead><tbody>'
            + rows.map((r) => {
                const key = r[1];
                const info = deltaKey[key] ? deltaInfo(d[deltaKey[key]], r[2]) : null;
                const badge = info ? `<span class="bl-delta ${info.cls}">${info.text}</span>` : '';
                return `<tr><td>${r[0]}</td><td>${r[3](cmp.a[key])}</td><td>${r[3](cmp.b[key])}</td><td>${badge}</td></tr>`;
            }).join('')
            + '</tbody></table>';

        applyDelta('dQps', d.qps_pct, false);
        applyDelta('dP95', d.p95_pct, true);
        applyDelta('dP99', d.p99_pct, true);
    }

    // ------------------------------------------------------------------
    // Controls state
    // ------------------------------------------------------------------

    function currentStep() {
        const opt = state.optimization;
        const applied = !!(opt && opt.applied);
        if (isRunning()) return applied ? 3 : 1;
        const last = state.slots.B || state.slots.A;
        if (!last) return 1;
        if (last.indexed) return 4;
        return applied ? 3 : 2;
    }

    function updateControls() {
        const running = isRunning();
        const opt = state.optimization;
        const step = currentStep();

        $('startBtn').disabled = running || state.starting || !state.scenarios.length;
        $('stopBtn').disabled = !running || state.stopping;
        $('stopBtn').textContent = state.stopping ? 'Stopping...' : 'Stop';
        ['scenarioSelect', 'threadsInput', 'durationInput'].forEach((id) => {
            $(id).disabled = running || state.starting || !state.scenarios.length;
        });
        $('applyBtn').disabled = running || state.indexBusy || !opt || opt.applied;
        $('revertBtn').disabled = running || state.indexBusy || !opt || !opt.applied;

        $('applyBtn').classList.toggle('is-hot', step === 2);
        $('startBtn').classList.toggle('is-hot', step === 1 && !running || step === 3 && !running);

        document.querySelectorAll('#steps li').forEach((li) => {
            const n = Number(li.dataset.step);
            li.classList.toggle('is-active', n === step);
            li.classList.toggle('is-done', n < step);
        });
    }

    function renderIndex() {
        const opt = state.optimization;
        const badge = $('idxState');
        if (!opt) {
            $('idxTitle').textContent = '-';
            $('idxStmt').textContent = '';
            badge.dataset.state = 'unknown';
            badge.textContent = 'unavailable';
            return;
        }
        $('idxTitle').textContent = opt.title || opt.name;
        const stmts = (opt.statements || []).join('; ');
        $('idxStmt').textContent = stmts;
        $('idxStmt').title = `${opt.description || ''}\n${stmts}`.trim();
        badge.dataset.state = opt.applied ? 'applied' : 'missing';
        badge.textContent = opt.applied ? 'Applied' : 'Not applied';
        const applyBtn = $('applyBtn');
        if (!state.indexBusy) applyBtn.textContent = 'Apply suggested index';
        if (!state.indexBusy) $('revertBtn').textContent = 'Revert';
    }

    // ------------------------------------------------------------------
    // Run lifecycle: start, stream, polling, finish
    // ------------------------------------------------------------------

    function readNumber(id, lo, hi, label) {
        const v = Number($(id).value);
        if (!Number.isFinite(v) || v < lo || v > hi) {
            throw new ApiError(`${label} must be a number between ${lo} and ${hi}.`);
        }
        return Math.round(v);
    }

    async function startRun() {
        if (isRunning() || state.starting || !state.scenarios.length) return;
        clearError();
        let threads, duration;
        try {
            threads = readNumber('threadsInput', 1, 256, 'Threads');
            duration = readNumber('durationInput', 2, 600, 'Duration');
        } catch (e) {
            showError(e.message);
            return;
        }
        state.starting = true;
        updateControls();
        try {
            // Refresh index state so the run is labelled correctly even if it changed elsewhere.
            try { state.optimization = await api('GET', API.optimization); renderIndex(); } catch (e) { /* keep cached state */ }
            const name = $('scenarioSelect').value;
            const data = await api('POST', API.run(name), { threads, duration_seconds: duration });
            state.ghost = state.slots.B || state.slots.A;
            state.current = {
                id: data.id, scenario: data.scenario || name, threads, duration,
                indexed: !!(state.optimization && state.optimization.applied),
                samples: [], lastSeq: 0, totalQueries: 0, totalErrors: 0,
                finished: false, es: null, pollTimer: null, errCount: 0, mode: 'sse',
            };
            state.cmp = null;
            resetKpis();
            Graph.reset();
            Charts.render();
            $('queryList').innerHTML = '<p class="bl-muted">Waiting for the first sample.</p>';
            $('progressBar').style.width = '0';
            setPill('live', 'Connecting...');
            connectStream(state.current);
        } catch (e) {
            showError(`Could not start the benchmark: ${e.message}`);
            setPill('error', 'Start failed');
        } finally {
            state.starting = false;
            updateControls();
        }
    }

    function connectStream(run) {
        let es;
        try {
            es = new EventSource(API.stream(run.id));
        } catch (e) {
            startPolling(run);
            return;
        }
        run.es = es;
        es.addEventListener('open', () => {
            run.errCount = 0;
            if (!run.finished) setPill('live', 'Live (SSE)');
        });
        es.addEventListener('sample', (ev) => {
            try { onSample(run, JSON.parse(ev.data)); } catch (e) { console.error('Bad sample event', e); }
        });
        es.addEventListener('status', (ev) => {
            if (run.finished) return;
            try {
                const st = JSON.parse(ev.data);
                if (st.status === 'running') setPill('live', st.phase ? `Live - ${st.phase}` : 'Live (SSE)');
            } catch (e) { /* ignore */ }
        });
        es.addEventListener('done', (ev) => {
            let d = {};
            try { d = JSON.parse(ev.data); } catch (e) { /* ignore */ }
            finishRun(run, d.status || 'completed', d.summary, d.error);
        });
        es.onerror = () => {
            if (run.finished) return;
            run.errCount += 1;
            if (es.readyState === EventSource.CLOSED || run.errCount >= 3) {
                es.close();
                startPolling(run);
            } else {
                setPill('poll', 'Reconnecting...');
            }
        };
    }

    function startPolling(run) {
        if (run.finished || run.mode === 'poll') return;
        run.mode = 'poll';
        if (run.es) { run.es.close(); run.es = null; }
        setPill('poll', 'Live (polling)');
        let failures = 0;
        const tick = async () => {
            if (run.finished) return;
            try {
                const d = await api('GET', API.samples(run.id, run.lastSeq));
                failures = 0;
                (d.samples || []).forEach((s) => onSample(run, s));
                if (d.status && d.status !== 'running') {
                    finishRun(run, d.status, null, '');
                    return;
                }
            } catch (e) {
                failures += 1;
                if (failures >= 5) {
                    finishRun(run, 'failed', null, `Lost connection to the benchmark: ${e.message}`);
                    return;
                }
            }
            run.pollTimer = setTimeout(tick, 1000);
        };
        tick();
    }

    function onSample(run, s) {
        if (run.finished || s.seq <= run.lastSeq) return;
        run.lastSeq = s.seq;
        run.errCount = 0;
        run.samples.push(s);
        run.totalQueries += (s.interval && s.interval.queries) || 0;
        run.totalErrors += (s.interval && s.interval.errors) || 0;
        if (run !== state.current) return;
        renderKpis(s);
        Charts.render();
        Graph.update(s.graph);
        renderQueries(s.queries);
        $('progressBar').style.width = `${clamp((s.elapsed_seconds / run.duration) * 100, 0, 100)}%`;
    }

    function summaryFromSamples(samples) {
        if (!samples.length) return { qps: 0, avg_latency_ms: 0, p50_latency_ms: 0, p95_latency_ms: 0, p99_latency_ms: 0, error_rate: 0, total_queries: 0, duration_seconds: 0 };
        const total = samples.reduce((s, x) => s + x.interval.queries, 0) || 1;
        const w = (k) => samples.reduce((s, x) => s + x.interval[k] * x.interval.queries, 0) / total;
        const dur = samples[samples.length - 1].elapsed_seconds || 1;
        const errors = samples.reduce((s, x) => s + x.interval.errors, 0);
        return {
            qps: total / dur, avg_latency_ms: w('avg_latency_ms'), p50_latency_ms: w('p50_latency_ms'),
            p95_latency_ms: w('p95_latency_ms'), p99_latency_ms: w('p99_latency_ms'),
            error_rate: errors / total, total_queries: total, duration_seconds: dur,
        };
    }

    async function finishRun(run, status, summary, error) {
        if (run.finished) return;
        run.finished = true;
        if (run.es) { run.es.close(); run.es = null; }
        clearTimeout(run.pollTimer);
        state.stopping = false;
        run.status = status;
        run.summary = summary || summaryFromSamples(run.samples);

        if (status === 'failed') {
            showError(error ? `Benchmark failed: ${error}` : 'Benchmark failed.');
            setPill('error', 'Failed');
        } else {
            setPill('done', status === 'cancelled' ? 'Stopped' : 'Completed');
            $('progressBar').style.width = status === 'completed' ? '100%' : $('progressBar').style.width;
            if (run.samples.length >= 3) storeRun(run);
        }
        updateControls();
        renderCompare();
        Charts.render();

        if (state.slots.A && state.slots.B && state.slots.B === run) {
            state.cmp = localCompare(state.slots.A, state.slots.B);
            renderCompare();
            try {
                const data = await api('GET', API.compare(state.slots.A.id, state.slots.B.id));
                if (state.slots.B === run) {
                    state.cmp = Object.assign({ source: 'api' }, data);
                    renderCompare();
                }
            } catch (e) {
                console.warn('Compare endpoint failed, using local delta:', e.message);
            }
            Charts.render();
        }
    }

    function storeRun(run) {
        if (!state.slots.A) state.slots.A = run;
        else if (!state.slots.B) state.slots.B = run;
        else { state.slots.A = state.slots.B; state.slots.B = run; }
    }

    async function stopRun() {
        const run = state.current;
        if (!isRunning() || state.stopping) return;
        state.stopping = true;
        updateControls();
        try {
            await api('POST', API.stop(run.id));
            // The terminal state arrives through the stream (done event) or polling.
        } catch (e) {
            state.stopping = false;
            showError(`Could not stop the benchmark: ${e.message}`);
            updateControls();
        }
    }

    // ------------------------------------------------------------------
    // Optimization (index) actions
    // ------------------------------------------------------------------

    async function setIndex(apply) {
        if (isRunning() || state.indexBusy) return;
        clearError();
        state.indexBusy = true;
        $(apply ? 'applyBtn' : 'revertBtn').textContent = apply ? 'Applying...' : 'Reverting...';
        updateControls();
        try {
            state.optimization = await api('POST', apply ? `${API.optimization}/apply` : `${API.optimization}/revert`);
        } catch (e) {
            showError(`Could not ${apply ? 'apply' : 'revert'} the index: ${e.message}`);
        } finally {
            state.indexBusy = false;
            renderIndex();
            updateControls();
        }
    }

    // ------------------------------------------------------------------
    // Presentation mode, theme, keyboard
    // ------------------------------------------------------------------

    function setPresentation(on) {
        document.body.classList.toggle('presentation', on);
        $('presentBtn').textContent = on ? 'Exit presentation' : 'Presentation';
        setTimeout(() => Charts.resize(), 80);
    }

    function togglePresentation() {
        const on = !document.body.classList.contains('presentation');
        setPresentation(on);
        const el = document.documentElement;
        if (on && !document.fullscreenElement && el.requestFullscreen) {
            el.requestFullscreen().catch(() => { /* presentation layout still applies */ });
        } else if (!on && document.fullscreenElement) {
            document.exitFullscreen().catch(() => {});
        }
    }

    function applyTheme(theme) {
        document.body.dataset.theme = theme;
        try { localStorage.setItem('theme', theme); } catch (e) { /* ignore */ }
        $('themeToggle').textContent = theme === 'dark' ? 'Light theme' : 'Dark theme';
        if (Charts) Charts.retheme();
        Graph.recolor();
    }

    function toggleTheme() {
        applyTheme(document.body.dataset.theme === 'dark' ? 'light' : 'dark');
    }

    function onKeyDown(ev) {
        if (ev.ctrlKey || ev.metaKey || ev.altKey) return;
        const tag = ev.target && ev.target.tagName;
        if (tag === 'INPUT' || tag === 'SELECT' || tag === 'TEXTAREA') return;
        const key = ev.key.toLowerCase();
        if (ev.key === ' ' || key === 'spacebar') {
            ev.preventDefault();
            if (ev.repeat) return;
            if (isRunning()) stopRun(); else startRun();
        } else if (key === 'p') {
            togglePresentation();
        } else if (key === 't') {
            toggleTheme();
        } else if (key === 'i') {
            const opt = state.optimization;
            if (opt) setIndex(!opt.applied);
        }
    }

    // Space on a focused button would also fire click on keyup; swallow it.
    function onKeyUp(ev) {
        const tag = ev.target && ev.target.tagName;
        if (ev.key === ' ' && tag !== 'INPUT' && tag !== 'SELECT' && tag !== 'TEXTAREA') ev.preventDefault();
    }

    // ------------------------------------------------------------------
    // Init
    // ------------------------------------------------------------------

    function onScenarioChange() {
        const sc = state.scenarios.find((s) => s.name === $('scenarioSelect').value);
        if (!sc) return;
        $('threadsInput').value = sc.threads || 8;
        $('durationInput').value = sc.duration_seconds || 30;
        $('scenarioSelect').title = sc.description || '';
    }

    async function loadScenarios() {
        state.scenarios = (await api('GET', API.scenarios)) || [];
        const sel = $('scenarioSelect');
        sel.innerHTML = state.scenarios.map((s) => `<option value="${esc(s.name)}">${esc(s.title || s.name)}</option>`).join('');
        if (!state.scenarios.length) throw new ApiError('No demo scenarios are configured on the server.');
        onScenarioChange();
    }

    async function loadTopology() {
        const topo = await api('GET', API.topology);
        if (!topo || !topo.nodes || !topo.nodes.length) throw new ApiError('The demo topology is empty.');
        Graph.init(topo);
        state.topologyReady = true;
    }

    async function loadOptimization() {
        state.optimization = await api('GET', API.optimization);
    }

    async function init() {
        let saved = 'dark';
        try { saved = localStorage.getItem('theme') || 'dark'; } catch (e) { /* ignore */ }
        Charts.init();
        applyTheme(saved === 'light' ? 'light' : 'dark');

        $('startBtn').addEventListener('click', (e) => { startRun(); e.currentTarget.blur(); });
        $('stopBtn').addEventListener('click', (e) => { stopRun(); e.currentTarget.blur(); });
        $('applyBtn').addEventListener('click', (e) => { setIndex(true); e.currentTarget.blur(); });
        $('revertBtn').addEventListener('click', (e) => { setIndex(false); e.currentTarget.blur(); });
        $('presentBtn').addEventListener('click', (e) => { togglePresentation(); e.currentTarget.blur(); });
        $('themeToggle').addEventListener('click', (e) => { toggleTheme(); e.currentTarget.blur(); });
        $('errorClose').addEventListener('click', clearError);
        $('scenarioSelect').addEventListener('change', onScenarioChange);
        document.addEventListener('keydown', onKeyDown);
        document.addEventListener('keyup', onKeyUp);
        document.addEventListener('fullscreenchange', () => {
            if (!document.fullscreenElement) setPresentation(false);
        });

        updateControls();
        renderIndex();

        const results = await Promise.allSettled([loadScenarios(), loadTopology(), loadOptimization()]);
        const labels = ['scenarios', 'topology', 'optimization'];
        const failed = results
            .map((r, i) => (r.status === 'rejected' ? `${labels[i]}: ${r.reason.message}` : null))
            .filter(Boolean);
        if (failed.length) {
            showError(`Could not load demo data (${failed.join('; ')}).`);
            if (!state.topologyReady) {
                const empty = $('graphEmpty');
                empty.hidden = false;
                empty.textContent = 'The table graph is unavailable until the demo topology can be loaded.';
            }
        }
        renderIndex();
        updateControls();
    }

    document.addEventListener('DOMContentLoaded', init);
})();
