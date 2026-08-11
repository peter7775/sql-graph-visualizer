/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

class GraphVisualizer {
    constructor() {
        this.viz = null;
        this.network = null;
        this.currentGraphNodes = [];
        this.currentGraphEdges = [];
        this.originalNodeVisuals = new Map();
        this.originalEdgeVisuals = new Map();
        this.performanceOverlayActive = false;
        this.performanceData = null;
        this.initialize();
    }

    async initialize() {
        console.log('Initializing visualization...');

        try {
            const graphResponse = await fetch('/api/graph');
            if (!graphResponse.ok) {
                throw new Error(`HTTP error! status: ${graphResponse.status}`);
            }
            const graphData = await graphResponse.json();
            console.log('Graph data loaded from API:', graphData);

            const container = document.getElementById('viz');
            if (!container) {
                throw new Error('Container "viz" was not found!');
            }
            console.log('Container found:', container);

            const nodes = new vis.DataSet();
            const edges = new vis.DataSet();

            this.currentGraphNodes = [];
            this.currentGraphEdges = [];
            this.originalNodeVisuals = new Map();
            this.originalEdgeVisuals = new Map();

            if (graphData.nodes) {
                graphData.nodes.forEach(node => {
                    const displayLabel = node.properties.name || 
                        node.properties.nazev || 
                        node.properties.title ||
                        node.properties.expert_name ||
                        node.properties.team_name ||
                        node.properties.skill_name ||
                        node.properties.php_code || 
                        node.properties.id || 
                        node.label || 
                        'N/A';
                    
                    const tooltip = Object.entries(node.properties)
                        .filter(([key, value]) => value != null && value !== '' && key !== 'name')
                        .map(([key, value]) => {
                            if (typeof value === 'string' && value.length > 50) {
                                return `${key}: ${value.substring(0, 47)}...`;
                            }
                            return `${key}: ${value}`;
                        })
                        .join('\n') + '\n\nType: ' + node.label;

                    let nodeSize = 25;
                    if (node.label === 'HighImpactProject') nodeSize = 40;
                    else if (node.label === 'Project') nodeSize = 35;
                    else if (node.label === 'Team' || node.label === 'TeamSummary') nodeSize = 30;
                    else if (node.label === 'User' || node.label === 'Skill') nodeSize = 25;
                    else if (node.label === 'Task') nodeSize = 20;
                    else if (node.label === 'SkillExpert') nodeSize = 30;

                    nodes.add({
                        id: node.id,
                        label: displayLabel.length > 20 ? displayLabel.substring(0, 17) + '...' : displayLabel,
                        title: tooltip,
                        group: node.label,
                        size: nodeSize,
                        properties: node.properties
                    });

                    this.currentGraphNodes.push({
                        id: node.id,
                        label: node.label,
                        properties: node.properties || {}
                    });
                });
            }

            if (graphData.relationships) {
                graphData.relationships.forEach((rel, relIndex) => {
                    let edgeColor = '#848484';
                    let edgeWidth = 2;
                    let edgeLabel = rel.type;
                    const edgeId = `edge-${relIndex}`;

                    switch(rel.type) {
                        case 'LEADS': edgeColor = '#D0021B'; edgeWidth = 3; break;
                        case 'MEMBER_OF': edgeColor = '#7ED321'; break;
                        case 'HAS_SKILL': edgeColor = '#50E3C2'; break;
                        case 'REQUIRES_SKILL': edgeColor = '#F5A623'; break;
                        case 'ASSIGNED_TO': edgeColor = '#BD10E0'; break;
                        case 'DEPENDS_ON': edgeColor = '#FF6B6B'; edgeWidth = 3; break;
                        case 'SUBTASK_OF': edgeColor = '#9013FE'; break;
                        case 'SKILL_COMPATIBLE': edgeColor = '#4A90E2'; edgeWidth = 1; break;
                        case 'EXPERT_IN': edgeColor = '#9013FE'; edgeWidth = 4; break;
                        case 'ENHANCED_VIEW_OF': edgeColor = '#FF9500'; edgeWidth = 3; break;
                    }

                    let edgeTooltip = `Relationship: ${rel.type}`;
                    if (rel.properties && Object.keys(rel.properties).length > 0) {
                        edgeTooltip += '\nProperties:\n' + 
                            Object.entries(rel.properties)
                                .map(([key, value]) => `  ${key}: ${value}`)
                                .join('\n');
                    }

                    edges.add({
                        id: edgeId,
                        from: rel.from,
                        to: rel.to,
                        label: edgeLabel,
                        title: edgeTooltip,
                        color: {
                            color: edgeColor,
                            highlight: '#FF6B6B'
                        },
                        width: edgeWidth,
                        arrows: { to: { enabled: true, scaleFactor: 0.8 } }
                    });

                    this.currentGraphEdges.push({
                        id: edgeId,
                        from: rel.from,
                        to: rel.to,
                        type: rel.type
                    });
                });
            }

            this.captureOriginalVisuals(nodes, edges);

            console.log('Processed nodes:', nodes.get().length);
            console.log('Processed edges:', edges.get().length);

            const options = this.getThemeOptions();
            const data = { nodes, edges };
            this.network = new vis.Network(container, data, options);

            console.log('Network created with', nodes.get().length, 'nodes and', edges.get().length, 'edges');

            this.network.on('stabilizationProgress', function(params) {
                console.log('Stabilization:', Math.round(params.iterations/params.total * 100), '%');
            });

            this.network.on('stabilizationIterationsDone', function() {
                console.log('Stabilization completed');
            });

            this.initializeEventListeners();

            this.applyThemeToNetwork();

            const perfToggle = document.getElementById('perfOverlayToggle');
            if (perfToggle && perfToggle.checked) {
                this.enablePerformanceOverlay();
            }

        } catch (error) {
            console.error('Error initializing visualization:', error);
        }
    }

    captureOriginalVisuals(nodes, edges) {
        nodes.get().forEach(n => {
            this.originalNodeVisuals.set(n.id, {
                size: n.size,
                color: n.color,
                borderWidth: n.borderWidth,
                shadow: n.shadow
            });
        });
        edges.get().forEach(e => {
            this.originalEdgeVisuals.set(e.id, {
                width: e.width,
                color: e.color
            });
        });
    }

    getThemeOptions() {
        const isDark = document.body.getAttribute("data-theme") === "dark";

        return {
            nodes: {
                shape: 'dot',
                size: 30,
                font: {
                    size: 12,
                    face: 'Arial',
                    vadjust: 0,
                    background: isDark ? 'rgba(34,34,34,0.85)' : 'rgba(255,255,255,0.8)',
                    color: isDark ? '#ffffff' : '#000000'
                },
                borderWidth: 2
            },
            edges: {
                arrows: { to: { enabled: true, scaleFactor: 0.8 } },
                font: {
                    size: 10,
                    align: 'middle',
                    background: isDark ? 'rgba(34,34,34,0.85)' : 'rgba(255,255,255,0.9)',
                    color: isDark ? '#ffffff' : '#000000',
                    strokeWidth: 0,
                },
                smooth: { type: 'dynamic', roundness: 0.2 }
            },
            physics: {
                enabled: true,
                solver: 'forceAtlas2Based'
            }
        };
    }

    initializeEventListeners() {
        if (!this.network) return;

        this.network.on('click', (params) => {
            if (params.nodes.length > 0) {
                console.log('Node clicked:', params.nodes[0]);
            }
        });

        this.initializeControlButtons();
        this.initializeSearch();
        this.initializeLayoutSelector();
        this.initializePerformanceOverlay();
    }

    ////////////////

    initializeControlButtons() {
        const zoomInBtn = document.getElementById('zoomIn');
        if (zoomInBtn) {
            zoomInBtn.addEventListener('click', () => {
                const currentScale = this.network.getScale();
                this.network.moveTo({
                    scale: currentScale * 1.2
                });
                console.log('Zoomed in to scale:', currentScale * 1.2);
            });
        }

        const zoomOutBtn = document.getElementById('zoomOut');
        if (zoomOutBtn) {
            zoomOutBtn.addEventListener('click', () => {
                const currentScale = this.network.getScale();
                this.network.moveTo({
                    scale: currentScale * 0.8
                });
                console.log('Zoomed out to scale:', currentScale * 0.8);
            });
        }

        const fitBtn = document.getElementById('fit');
        if (fitBtn) {
            fitBtn.addEventListener('click', () => {
                this.network.fit({
                    animation: {
                        duration: 1000,
                        easingFunction: 'easeInOutQuad'
                    }
                });
                console.log('Fitted all nodes into view');
            });
        }

        const reloadBtn = document.getElementById('reload');
        if (reloadBtn) {
            reloadBtn.addEventListener('click', async () => {
                console.log('Reloading graph data...');
                reloadBtn.disabled = true;
                reloadBtn.textContent = 'Loading...';
                
                try {
                    await this.initialize();
                    console.log('Graph data reloaded successfully');
                } catch (error) {
                    console.error('Error reloading graph:', error);
                    alert('Error reloading graph data. Please try again.');
                } finally {
                    reloadBtn.disabled = false;
                    reloadBtn.textContent = 'Reload';
                }
            });
        }
    }

    initializeLayoutSelector() {
        const layoutSelector = document.getElementById('layout');
        if (layoutSelector) {
            layoutSelector.addEventListener('change', (event) => {
                const selectedLayout = event.target.value;
                this.applyLayout(selectedLayout);
            });
        }
    }

    applyLayout(layoutType) {
        if (!this.network) return;

        let options = {};
        
        switch(layoutType) {
            case 'hierarchical':
                options = {
                    layout: {
                        hierarchical: {
                            enabled: true,
                            direction: 'UD',
                            sortMethod: 'hubsize',
                            shakeTowards: 'roots',
                            levelSeparation: 150,
                            nodeSpacing: 100
                        }
                    },
                    physics: {
                        enabled: false
                    }
                };
                break;
            
            case 'circular':
                options = {
                    layout: {
                        hierarchical: {
                            enabled: false
                        }
                    },
                    physics: {
                        enabled: false
                    }
                };
                
                setTimeout(() => {
                    this.arrangeNodesCircularly();
                }, 100);
                break;
            
            case 'force':
            default:
                options = {
                    layout: {
                        hierarchical: {
                            enabled: false
                        }
                    },
                    physics: {
                        enabled: true,
                        solver: 'forceAtlas2Based',
                        forceAtlas2Based: {
                            gravitationalConstant: -80,
                            centralGravity: 0.02,
                            springLength: 150,
                            springConstant: 0.05,
                            damping: 0.4,
                            avoidOverlap: 0.1
                        },
                        stabilization: {
                            enabled: true,
                            iterations: 1000
                        }
                    }
                };
                break;
        }

        this.network.setOptions(options);
        console.log('Applied layout:', layoutType);

        setTimeout(() => {
            this.network.fit({
                animation: {
                    duration: 1000,
                    easingFunction: 'easeInOutQuad'
                }
            });
        }, layoutType === 'force' ? 2000 : 500);
    }

    arrangeNodesCircularly() {
        if (!this.network) return;

        const nodes = this.network.body.data.nodes.get();
        const nodeCount = nodes.length;
        const radius = Math.max(200, nodeCount * 10);
        const centerX = 0;
        const centerY = 0;

        const updatePositions = [];
        
        nodes.forEach((node, index) => {
            const angle = (2 * Math.PI * index) / nodeCount;
            const x = centerX + radius * Math.cos(angle);
            const y = centerY + radius * Math.sin(angle);
            
            updatePositions.push({
                id: node.id,
                x: x,
                y: y
            });
        });

        this.network.body.data.nodes.update(updatePositions);
        console.log('Arranged', nodeCount, 'nodes in circular layout with radius', radius);
    }

    initializeSearch() {
        const searchInput = document.getElementById('search');
        const searchResults = document.getElementById('searchResults');
        
        if (!searchInput || !searchResults) return;

        let searchTimeout;
        
        searchInput.addEventListener('input', (event) => {
            clearTimeout(searchTimeout);
            
            const query = event.target.value.trim().toLowerCase();
            
            if (query.length < 2) {
                searchResults.style.display = 'none';
                return;
            }
            
            searchTimeout = setTimeout(() => {
                this.performSearch(query, searchResults);
            }, 300);
        });

        document.addEventListener('click', (event) => {
            if (!searchInput.contains(event.target) && !searchResults.contains(event.target)) {
                searchResults.style.display = 'none';
            }
        });
    }

    performSearch(query, resultsContainer) {
        if (!this.network) return;

        const nodes = this.network.body.data.nodes.get();
        const matchedNodes = [];
        
        nodes.forEach(node => {
            if (node.label && node.label.toLowerCase().includes(query)) {
                matchedNodes.push({
                    node: node,
                    matchType: 'label',
                    matchText: node.label
                });
                return;
            }
            
            if (node.properties) {
                for (const [key, value] of Object.entries(node.properties)) {
                    if (value && value.toString().toLowerCase().includes(query)) {
                        matchedNodes.push({
                            node: node,
                            matchType: 'property',
                            matchText: `${key}: ${value}`
                        });
                        break;
                    }
                }
            }
        });
        
        this.displaySearchResults(matchedNodes.slice(0, 10), resultsContainer);
    }

    displaySearchResults(matches, container) {
        container.innerHTML = '';
        
        if (matches.length === 0) {
            container.innerHTML = '<div class="search-result-item">No results found</div>';
        } else {
            matches.forEach(match => {
                const item = document.createElement('div');
                item.className = 'search-result-item';
                item.innerHTML = `
                    <strong>${match.node.label}</strong><br>
                    <small class="text-muted">${match.matchText}</small>
                `;
                
                item.addEventListener('click', () => {
                    this.focusOnNode(match.node.id);
                    container.style.display = 'none';
                    document.getElementById('search').value = match.node.label;
                });
                
                container.appendChild(item);
            });
        }
        
        container.style.display = 'block';
    }

    focusOnNode(nodeId) {
        if (!this.network) return;

        this.network.selectNodes([nodeId]);
        
        this.network.focus(nodeId, {
            scale: 1.5,
            animation: {
                duration: 1000,
                easingFunction: 'easeInOutQuad'
            }
        });
        
        console.log('Focused on node:', nodeId);
    }

    applyThemeToNetwork() {
        if (!this.network) return;
        const isDark = document.body.getAttribute("data-theme") === "dark";

        this.network.setOptions({
            nodes: {
                font: {
                    background: isDark ? '#222222' : '#ffffff',
                    color: isDark ? '#ffffff' : '#000000'
                }
            },
            edges: {
                font: {
                    background: isDark ? '#222222' : '#ffffff',
                    color: isDark ? '#ffffff' : '#000000'
                }
            }
        });

        const container = this.network.body.container;
        container.style.backgroundColor = isDark ? '#121212' : '#ffffff';
    }

    ////////////////
    // Performance overlay
    //
    // The performance overlay fetches /api/performance/data/graph and maps the
    // returned performance metrics onto the existing vis-network nodes/edges
    // (size/color/border for nodes, thickness/color for edges), without
    // rebuilding the graph. Matching between the domain graph (/api/graph,
    // Neo4j-based ids) and the performance graph (table/label-derived ids) is
    // best-effort; any node/edge that cannot be confidently matched is simply
    // left with its original appearance (graceful no-op, no thrown errors).

    initializePerformanceOverlay() {
        const toggle = document.getElementById('perfOverlayToggle');
        const select = document.getElementById('perfMetricSelect');
        if (!toggle) return;

        toggle.addEventListener('change', () => {
            if (toggle.checked) {
                if (select) select.disabled = false;
                this.enablePerformanceOverlay();
            } else {
                this.disablePerformanceOverlay();
                if (select) select.disabled = true;
            }
        });

        if (select) {
            select.addEventListener('change', () => {
                if (toggle.checked && this.performanceData) {
                    this.applyPerformanceOverlay(select.value);
                }
            });
        }
    }

    getSelectedPerformanceMetric() {
        const select = document.getElementById('perfMetricSelect');
        return select ? select.value : 'average_latency';
    }

    setPerformanceStatus(message) {
        const el = document.getElementById('perfOverlayStatus');
        if (el) el.textContent = message || '';
    }

    async enablePerformanceOverlay() {
        this.setPerformanceStatus('Loading performance data...');
        try {
            const perfData = await this.fetchPerformanceGraphData();
            this.performanceData = perfData;
            this.performanceOverlayActive = true;
            this.applyPerformanceOverlay(this.getSelectedPerformanceMetric());
        } catch (error) {
            console.warn('Performance overlay unavailable:', error.message);
            this.performanceOverlayActive = false;
            this.performanceData = null;
            this.setPerformanceStatus('Performance data unavailable');

            const toggle = document.getElementById('perfOverlayToggle');
            if (toggle) toggle.checked = false;
            const select = document.getElementById('perfMetricSelect');
            if (select) select.disabled = true;
        }
    }

    disablePerformanceOverlay() {
        if (!this.network) return;

        const nodesDS = this.network.body.data.nodes;
        const edgesDS = this.network.body.data.edges;

        const nodeUpdates = [];
        this.originalNodeVisuals.forEach((visual, id) => {
            if (!nodesDS.get(id)) return;
            nodeUpdates.push({
                id,
                size: visual.size,
                color: visual.color,
                borderWidth: visual.borderWidth,
                shadow: visual.shadow
            });
        });
        if (nodeUpdates.length) nodesDS.update(nodeUpdates);

        const edgeUpdates = [];
        this.originalEdgeVisuals.forEach((visual, id) => {
            if (!edgesDS.get(id)) return;
            edgeUpdates.push({
                id,
                width: visual.width,
                color: visual.color
            });
        });
        if (edgeUpdates.length) edgesDS.update(edgeUpdates);

        this.performanceOverlayActive = false;
        this.performanceData = null;
        this.setPerformanceStatus('');
    }

    // Builds the list of candidate URLs to try for the performance API.
    // /api/performance/* is registered on the API server, which in local
    // development runs on a different port (default 8080) than the
    // visualization server (default 3000) that serves this page. In
    // single-port deployments (e.g. Railway) the relative path works directly.
    getPerformanceApiCandidates(path) {
        const candidates = [];

        if (window.PERFORMANCE_API_BASE_URL) {
            candidates.push(window.PERFORMANCE_API_BASE_URL.replace(/\/$/, '') + path);
        }

        candidates.push(path);

        if (window.location.port && window.location.port !== '8080') {
            candidates.push(`${window.location.protocol}//${window.location.hostname}:8080${path}`);
        }

        return [...new Set(candidates)];
    }

    async fetchPerformanceGraphData() {
        const path = '/api/performance/data/graph';
        const candidates = this.getPerformanceApiCandidates(path);
        let lastError = null;

        for (const url of candidates) {
            try {
                const response = await fetch(url);
                if (!response.ok) {
                    lastError = new Error(`HTTP ${response.status} from ${url}`);
                    continue;
                }

                const contentType = response.headers.get('content-type') || '';
                if (!contentType.includes('application/json')) {
                    lastError = new Error(`Unexpected content-type from ${url}`);
                    continue;
                }

                const payload = await response.json();
                if (!payload || payload.success !== true || !payload.data) {
                    lastError = new Error(`Unsuccessful response from ${url}`);
                    continue;
                }

                return payload.data;
            } catch (error) {
                lastError = error;
            }
        }

        throw lastError || new Error('No performance API endpoint reachable');
    }

    buildPerformanceIndexes(perfData) {
        const byId = new Map();
        const byTableName = new Map();
        const byLabel = new Map();

        (perfData.nodes || []).forEach(node => {
            if (node.id !== undefined && node.id !== null) {
                byId.set(String(node.id), node);
            }
            if (node.table_name) {
                if (!byTableName.has(node.table_name)) byTableName.set(node.table_name, []);
                byTableName.get(node.table_name).push(node);
            }
            if (node.label) {
                if (!byLabel.has(node.label)) byLabel.set(node.label, []);
                byLabel.get(node.label).push(node);
            }
        });

        return { byId, byTableName, byLabel };
    }

    // Best-effort matching between a domain graph node (/api/graph) and a
    // performance graph node (/api/performance/data/graph). Priority:
    // 1) explicit "id" property on the domain node, matched against perf node id
    // 2) domain node's "table_name" property, matched against perf table_name
    // 3) domain node label used as table name (mirrors backend default when no
    //    table_name property is set)
    // 4) domain node label matched uniquely against a single perf node's label
    // Returns null (no throw) when no confident match is found.
    matchPerformanceNode(domainNode, indexes) {
        const props = domainNode.properties || {};

        if (props.id !== undefined && props.id !== null) {
            const match = indexes.byId.get(String(props.id));
            if (match) return match;
        }

        if (props.table_name && indexes.byTableName.has(props.table_name)) {
            const candidates = indexes.byTableName.get(props.table_name);
            if (candidates.length === 1) return candidates[0];
        }

        if (domainNode.label && indexes.byTableName.has(domainNode.label)) {
            const candidates = indexes.byTableName.get(domainNode.label);
            if (candidates.length === 1) return candidates[0];
        }

        if (domainNode.label && indexes.byLabel.has(domainNode.label)) {
            const candidates = indexes.byLabel.get(domainNode.label);
            if (candidates.length === 1) return candidates[0];
        }

        return null;
    }

    buildNodeIdTranslationMap(domainNodes, indexes) {
        const map = new Map();
        domainNodes.forEach(domainNode => {
            const perfNode = this.matchPerformanceNode(domainNode, indexes);
            if (perfNode) map.set(String(domainNode.id), perfNode);
        });
        return map;
    }

    buildPerfEdgeIndex(perfEdges) {
        const map = new Map();
        (perfEdges || []).forEach(edge => {
            if (!edge.source_id || !edge.target_id) return;
            map.set(`${edge.source_id}|${edge.target_id}`, edge);
            map.set(`${edge.target_id}|${edge.source_id}`, edge);
        });
        return map;
    }

    matchPerformanceEdge(domainEdge, nodeIdMap, perfEdgeIndex) {
        const fromPerf = nodeIdMap.get(String(domainEdge.from));
        const toPerf = nodeIdMap.get(String(domainEdge.to));
        if (!fromPerf || !toPerf) return null;
        return perfEdgeIndex.get(`${fromPerf.id}|${toPerf.id}`) || null;
    }

    getNodeMetricValue(perfNode, metric) {
        const perf = perfNode.performance || {};
        switch (metric) {
            case 'queries_per_second': return perf.queries_per_second || 0;
            case 'hotspot_score': return perf.hotspot_score || 0;
            case 'load_score': return perf.load_score || 0;
            case 'average_latency':
            default: return perf.average_latency || 0;
        }
    }

    normalize(value, min, max) {
        if (max <= min) return 0;
        return Math.min(1, Math.max(0, (value - min) / (max - min)));
    }

    scaleRange(normalized, min, max) {
        return min + normalized * (max - min);
    }

    metricToColor(normalized) {
        if (normalized > 0.75) return '#f44336';
        if (normalized > 0.5) return '#ff9800';
        if (normalized > 0.25) return '#ffeb3b';
        return '#4caf50';
    }

    edgeMetricToColor(perf) {
        const rankColors = {
            critical: '#f44336',
            poor: '#f44336',
            fair: '#ff9800',
            good: '#ffeb3b',
            excellent: '#4caf50'
        };
        const rank = (perf && perf.performance_rank || '').toLowerCase();
        if (rank && rankColors[rank]) return rankColors[rank];

        const latency = (perf && perf.average_latency) || 0;
        if (latency > 500) return '#f44336';
        if (latency > 200) return '#ff9800';
        if (latency > 100) return '#ffeb3b';
        return '#4caf50';
    }

    applyPerformanceOverlay(metric) {
        if (!this.network || !this.performanceData) return;

        const nodesDS = this.network.body.data.nodes;
        const edgesDS = this.network.body.data.edges;

        const perfIndexes = this.buildPerformanceIndexes(this.performanceData);
        const nodeIdMap = this.buildNodeIdTranslationMap(this.currentGraphNodes, perfIndexes);
        const hotspotIds = new Set((this.performanceData.hotspots || []).map(h => h.node_id));

        const perfNodeValues = (this.performanceData.nodes || [])
            .map(node => this.getNodeMetricValue(node, metric))
            .filter(value => typeof value === 'number' && !Number.isNaN(value));
        const minVal = perfNodeValues.length ? Math.min(...perfNodeValues) : 0;
        const maxVal = perfNodeValues.length ? Math.max(...perfNodeValues) : 1;

        const nodeUpdates = [];
        this.currentGraphNodes.forEach(domainNode => {
            const perfNode = nodeIdMap.get(String(domainNode.id));
            if (!perfNode || !this.originalNodeVisuals.has(domainNode.id)) return;

            const value = this.getNodeMetricValue(perfNode, metric);
            const normalized = this.normalize(value, minVal, maxVal);
            const size = this.scaleRange(normalized, 15, 50);
            const color = this.metricToColor(normalized);
            const isHotspot = hotspotIds.has(perfNode.id);

            nodeUpdates.push({
                id: domainNode.id,
                size,
                color: {
                    background: color,
                    border: isHotspot ? '#d50000' : color,
                    highlight: { background: color, border: '#d50000' }
                },
                borderWidth: isHotspot ? 4 : 2,
                shadow: isHotspot
                    ? { enabled: true, color: 'rgba(213,0,0,0.6)', size: 15 }
                    : { enabled: false }
            });
        });
        if (nodeUpdates.length) nodesDS.update(nodeUpdates);

        const perfEdgeIndex = this.buildPerfEdgeIndex(this.performanceData.edges);
        const freqValues = (this.performanceData.edges || [])
            .map(edge => (edge.performance && edge.performance.query_frequency) || 0);
        const minFreq = freqValues.length ? Math.min(...freqValues) : 0;
        const maxFreq = freqValues.length ? Math.max(...freqValues) : 1;

        const edgeUpdates = [];
        this.currentGraphEdges.forEach(domainEdge => {
            const perfEdge = this.matchPerformanceEdge(domainEdge, nodeIdMap, perfEdgeIndex);
            if (!perfEdge || !this.originalEdgeVisuals.has(domainEdge.id)) return;

            const freq = (perfEdge.performance && perfEdge.performance.query_frequency) || 0;
            const width = this.scaleRange(this.normalize(freq, minFreq, maxFreq), 1, 10);
            const color = this.edgeMetricToColor(perfEdge.performance);

            edgeUpdates.push({
                id: domainEdge.id,
                width,
                color: { color, highlight: '#FF6B6B' }
            });
        });
        if (edgeUpdates.length) edgesDS.update(edgeUpdates);

        this.setPerformanceStatus(
            `Overlay active (${nodeUpdates.length}/${this.currentGraphNodes.length} nodes, ` +
            `${edgeUpdates.length}/${this.currentGraphEdges.length} edges matched)`
        );
    }
}

function initThemeToggle() {
    const body = document.body;
    const toggleBtn = document.getElementById("themeToggle");

    const savedTheme = localStorage.getItem("theme");
    if (savedTheme) {
        body.setAttribute("data-theme", savedTheme);
        toggleBtn.textContent = savedTheme === "dark" ? "☀️" : "🌙";
    }

    toggleBtn.addEventListener("click", () => {
        const currentTheme = body.getAttribute("data-theme");
        const newTheme = currentTheme === "light" ? "dark" : "light";
        body.setAttribute("data-theme", newTheme);
        localStorage.setItem("theme", newTheme);
        toggleBtn.textContent = newTheme === "dark" ? "☀️" : "🌙";

        if (window.graphVisualizer) {
            window.graphVisualizer.applyThemeToNetwork();
        }
    });
}

window.addEventListener("DOMContentLoaded", () => {
    window.graphVisualizer = new GraphVisualizer();
    initThemeToggle();
});
