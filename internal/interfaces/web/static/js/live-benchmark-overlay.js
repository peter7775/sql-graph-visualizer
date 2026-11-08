/* Explicit label/table and relationship/edge mapping for the instance graph. */
(function () {
    'use strict';
    window.LiveBenchmarkOverlay = class {
        constructor(visualizer) {
            this.visualizer = visualizer;
            this.toggle = document.getElementById('liveBenchToggle');
            this.status = document.getElementById('liveBenchStatus');
            this.generation = 0;
            if (!this.toggle) return;
            // Nodes may inherit their colour from a vis-network group rather
            // than carry it in the DataSet. Capture the resolved appearance
            // before applying the first live sample so disable restores it.
            const renderedNodes = visualizer.network && visualizer.network.body.nodes;
            visualizer.originalNodeVisuals.forEach((original, id) => {
                const options = renderedNodes && renderedNodes[id] && renderedNodes[id].options;
                if (!options || original.color != null) return;
                const color = options.color;
                if (color) original.color = {
                    background: color.background, border: color.border,
                    highlight: color.highlight && { background: color.highlight.background, border: color.highlight.border },
                    hover: color.hover && { background: color.hover.background, border: color.hover.border },
                };
            });
            this.toggle.onchange = () => this.toggle.checked ? this.enable() : this.disable();
            window.addEventListener('pagehide', () => this.disable());
        }

        async get(path) {
            const response = await fetch(path);
            if (!response.ok) throw new Error(`HTTP ${response.status}`);
            const body = await response.json();
            if (!body.success) throw new Error('Benchmark API unavailable');
            return body.data;
        }

        async enable() {
            const generation = ++this.generation;
            this.status.textContent = 'Loading benchmark data…';
            try {
                this.mapping = await this.get('/api/performance/live/mapping');
                if (generation !== this.generation) return;
                this.unsubscribe = window.LiveBenchmarkFeed.subscribe((sample) => {
                    if (generation !== this.generation) return;
                    this.liveReceived = true;
                    this.apply(sample);
                });
                const history = await this.get('/api/performance/benchmarks/history?limit=20');
                if (generation !== this.generation) return;
                const latest = (history.runs || []).find((run) => run.has_samples || run.status === 'running');
                if (!latest && this.liveReceived) return;
                if (!latest) throw new Error('No benchmark data');
                const data = await this.get(`/api/performance/benchmarks/${encodeURIComponent(latest.id)}/samples?since=0`);
                if (generation !== this.generation) return;
                if (!this.liveReceived && data.samples && data.samples.length) this.apply(data.samples[data.samples.length - 1]);
                if (!this.liveReceived && !(data.samples || []).length && data.status !== 'running') throw new Error('No samples');
                if (data.status === 'running' && !this.liveReceived) {
                    this.streamRunId = latest.id;
                    this.stream = new EventSource(`/api/performance/benchmarks/${encodeURIComponent(latest.id)}/stream`);
                    this.stream.addEventListener('sample', (event) => {
                        if (generation !== this.generation) return;
                        try { this.apply(JSON.parse(event.data)); } catch (e) { /* malformed sample */ }
                    });
                    this.stream.addEventListener('done', () => { if (this.stream) this.stream.close(); });
                    this.stream.onerror = () => { /* WebSocket remains the live source */ };
                }
            } catch (error) {
                if (generation !== this.generation) return;
                this.disable();
                this.status.textContent = 'No benchmark data yet - run one on /benchmark-live';
            }
        }

        apply(sample) {
            if (!this.toggle.checked || !this.mapping || !this.visualizer.network) return;
            if (this.lastSample && this.lastSample.benchmark_id === sample.benchmark_id && sample.seq <= this.lastSample.seq) return;
            if (this.stream && this.streamRunId !== sample.benchmark_id) {
                this.stream.close();
                this.stream = null;
            }
            this.lastSample = { benchmark_id: sample.benchmark_id, seq: sample.seq };
            const graph = sample.graph || {};
            const nodes = new Map((graph.nodes || []).map((n) => [n.id, n]));
            const edges = new Map((graph.edges || []).map((e) => [e.id, e]));
            const maxQps = Math.max(1, ...(graph.nodes || []).map((n) => n.qps || 0));
            const nodeUpdates = [], edgeUpdates = [];
            this.restoreVisuals();
            this.visualizer.currentGraphNodes.forEach((domain) => {
                const mapping = (this.mapping.labels || {})[domain.label];
                const metric = mapping && nodes.get(mapping.table);
                if (!metric) return;
                const color = window.LiveGraph.heatColor(metric.heat || 0);
                nodeUpdates.push({
                    id: domain.id, color: { background: color, border: color, highlight: { background: color, border: color } },
                    size: 20 + 30 * Math.sqrt(Math.min(1, Math.max(0, metric.qps / maxQps))),
                });
            });
            this.visualizer.currentGraphEdges.forEach((domain) => {
                const mapping = (this.mapping.relationships || {})[domain.type];
                if (!mapping) return;
                const ids = Array.isArray(mapping.edges) && mapping.edges.length ? mapping.edges : [mapping.edge];
                const metrics = ids.map((id) => edges.get(id)).filter(Boolean);
                if (!metrics.length) return;
                const latency = Math.max(0, ...metrics.map((metric) => metric.avg_latency_ms || 0));
                const load = Math.max(0, ...metrics.map((metric) => metric.load || 0));
                const color = window.LiveGraph.heatColor(window.LiveGraph.logHeat(latency));
                edgeUpdates.push({
                    id: domain.id, color: { color, highlight: color },
                    width: 3 + 11 * Math.pow(Math.min(1, load), 0.9),
                });
            });
            const data = this.visualizer.network.body.data;
            if (nodeUpdates.length) data.nodes.update(nodeUpdates);
            if (edgeUpdates.length) data.edges.update(edgeUpdates);
            this.status.textContent = `Benchmark ${sample.benchmark_id.slice(0, 8)} · ${nodeUpdates.length} nodes / ${edgeUpdates.length} edges`;
        }

        disable() {
            ++this.generation;
            if (this.unsubscribe) this.unsubscribe();
            if (this.stream) this.stream.close();
            this.unsubscribe = null;
            this.stream = null;
            this.liveReceived = false;
            this.lastSample = null;
            this.streamRunId = null;
            this.toggle.checked = false;
            this.status.textContent = '';
            this.restoreVisuals();
            const viz = this.visualizer;
            if (viz.performanceOverlayActive) viz.applyPerformanceOverlay(viz.getSelectedPerformanceMetric());
        }

        restoreVisuals() {
            const viz = this.visualizer;
            if (!viz.network) return;
            const data = viz.network.body.data;
            const nodes = [], edges = [];
            viz.originalNodeVisuals.forEach((original, id) => {
                nodes.push({ id, size: original.size, color: original.color || null,
                    borderWidth: original.borderWidth == null ? 2 : original.borderWidth, shadow: original.shadow || false });
            });
            viz.originalEdgeVisuals.forEach((original, id) => edges.push({ id, width: original.width, color: original.color }));
            if (nodes.length) data.nodes.update(nodes);
            if (edges.length) data.edges.update(edges);
        }
    };
})();
