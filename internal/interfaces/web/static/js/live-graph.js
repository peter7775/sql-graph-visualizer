/* Shared, isolated table-level live benchmark graph renderer. */
(function () {
    'use strict';
    const MAX_PARTICLES = 12;
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

    function create(element) {
        const host = typeof element === 'string' ? document.querySelector(element) : element;
        if (!host) throw new Error('Live graph SVG not found');
        const tooltip = document.createElement('div');
        tooltip.className = 'live-graph-tooltip';
        tooltip.hidden = true;
        host.parentElement.appendChild(tooltip);
        if (getComputedStyle(host.parentElement).position === 'static') host.parentElement.style.position = 'relative';
        const cssVar = (name) => getComputedStyle(host).getPropertyValue(name).trim()
            || (name === '--neutral-node' ? '#66758a' : '#8b98a9');
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
            maxQps = 1;
            hideTip();
            svg = d3.select(element).classed('live-graph', true);
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
                    id: e.id || [e.source, e.target].sort().join('-'), relation: e.relation || '',
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
            if (!best) return;
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
            nodes.forEach((n) => { n.active = false; n.qps = 0; });
            edges.forEach((e) => { e.active = false; e.qps = 0; });
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
            maxQps = 1;
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
            const tip = tooltip;
            tip.innerHTML = tipHtml(h);
            tip.hidden = false;
            const wrap = svg.node().parentElement;
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
            const tip = tooltip;
            if (!tip.hidden) tip.innerHTML = tipHtml(hover);
        }

        function hideTip() {
            hover = null;
            tooltip.hidden = true;
        }

        function destroy() {
            if (rafId) cancelAnimationFrame(rafId);
            rafId = null;
            if (svg) { svg.selectAll('*').interrupt('u'); svg.selectAll('*').remove(); }
            tooltip.remove();
        }
        return { init, update, reset, recolor: () => nodes.length && render(false), destroy };
    }

    window.LiveGraph = { create, heatColor, logHeat };
})();
