/**
 * Performance Graph Visualization Components
 * Enhanced visualization system with real-time performance data rendering,
 * heatmaps, animations, and interactive elements
 */

class PerformanceGraphVisualizer {
    constructor(containerId, options = {}) {
        this.containerId = containerId;
        this.container = document.getElementById(containerId);
        this.options = {
            width: options.width || 1200,
            height: options.height || 800,
            enableAnimations: options.enableAnimations !== false,
            enableHeatmap: options.enableHeatmap !== false,
            enableRealtime: options.enableRealtime !== false,
            updateInterval: options.updateInterval || 5000,
            maxNodes: options.maxNodes || 100,
            ...options
        };

        this.currentData = null;
        this.previousData = null;
        this.animationFrameId = null;
        this.websocket = null;
        
        this.initializeVisualization();
        this.setupEventHandlers();
        
        if (this.options.enableRealtime) {
            this.connectWebSocket();
        }
    }

    initializeVisualization() {
        // Create main visualization container
        this.svg = d3.select(`#${this.containerId}`)
            .append('svg')
            .attr('width', this.options.width)
            .attr('height', this.options.height)
            .attr('class', 'performance-graph-svg');

        // Create zoom and pan behavior
        this.zoom = d3.zoom()
            .scaleExtent([0.1, 10])
            .on('zoom', (event) => {
                this.mainGroup.attr('transform', event.transform);
            });

        this.svg.call(this.zoom);

        // Create main group for graph elements
        this.mainGroup = this.svg.append('g')
            .attr('class', 'main-group');

        // Create layers for different visualization elements
        this.createVisualizationLayers();
        
        // Initialize force simulation
        this.initializeForceSimulation();
        
        // Create performance overlay components
        this.createPerformanceOverlays();
        
        // Initialize color scales for performance mapping
        this.initializeColorScales();
    }

    createVisualizationLayers() {
        // Background layer for heatmaps and effects
        this.backgroundLayer = this.mainGroup.append('g')
            .attr('class', 'background-layer');

        // Edge layer for relationships
        this.edgeLayer = this.mainGroup.append('g')
            .attr('class', 'edge-layer');

        // Node layer for database tables
        this.nodeLayer = this.mainGroup.append('g')
            .attr('class', 'node-layer');

        // Animation layer for data flow particles
        this.animationLayer = this.mainGroup.append('g')
            .attr('class', 'animation-layer');

        // Overlay layer for labels and UI elements
        this.overlayLayer = this.mainGroup.append('g')
            .attr('class', 'overlay-layer');

        // Performance metrics layer
        this.metricsLayer = this.mainGroup.append('g')
            .attr('class', 'metrics-layer');
    }

    initializeForceSimulation() {
        this.simulation = d3.forceSimulation()
            .force('link', d3.forceLink().id(d => d.id).distance(100))
            .force('charge', d3.forceManyBody().strength(-300))
            .force('center', d3.forceCenter(this.options.width / 2, this.options.height / 2))
            .force('collision', d3.forceCollide().radius(d => d.size || 20))
            .on('tick', () => this.updateNodePositions());
    }

    createPerformanceOverlays() {
        // Performance legend
        this.createPerformanceLegend();
        
        // Real-time metrics panel
        this.createMetricsPanel();
        
        // Alert notifications area
        this.createAlertPanel();
        
        // Performance controls
        this.createControlPanel();
    }

    initializeColorScales() {
        // Color scales for different performance metrics
        this.colorScales = {
            latency: d3.scaleSequential(d3.interpolateRdYlGn).domain([1000, 0]),
            throughput: d3.scaleSequential(d3.interpolateBlues).domain([0, 1000]),
            errorRate: d3.scaleSequential(d3.interpolateReds).domain([0, 10]),
            hotspot: d3.scaleSequential(d3.interpolateOrRd).domain([0, 100])
        };

        this.sizeScale = d3.scaleLinear()
            .domain([0, 1000])
            .range([15, 50]);

        this.thicknessScale = d3.scaleLinear()
            .domain([0, 100])
            .range([1, 10]);
    }

    connectWebSocket() {
        const wsUrl = `ws://${window.location.host}/ws/performance`;
        this.websocket = new WebSocket(wsUrl);

        this.websocket.onopen = () => {
            console.log('WebSocket connected for performance data');
            this.showStatus('Connected to real-time performance .monitoring', 'success');
        };

        this.websocket.onmessage = (event) => {
            const message = JSON.parse(event.data);
            this.handleWebSocketMessage(message);
        };

        this.websocket.onclose = () => {
            console.log('WebSocket connection closed');
            this.showStatus('Real-time connection lost. Attempting to reconnect...', 'warning');
            
            // Attempt to reconnect after 5 seconds
            setTimeout(() => this.connectWebSocket(), 5000);
        };

        this.websocket.onerror = (error) => {
            console.error('WebSocket error:', error);
            this.showStatus('Connection error occurred', 'error');
        };
    }

    handleWebSocketMessage(message) {
        switch (message.topic) {
            case 'performance':
                if (message.data && message.data.nodes) {
                    this.updatePerformanceData(message.data);
                }
                break;
            case 'metrics':
                if (message.data) {
                    this.updateMetricsPanel(message.data);
                }
                break;
            case 'alerts':
                if (message.data) {
                    this.showAlert(message.data);
                }
                break;
        }
    }

    updatePerformanceData(data) {
        this.previousData = this.currentData;
        this.currentData = data;
        
        this.renderGraph();
        
        if (this.options.enableAnimations && this.previousData) {
            this.animateDataChanges();
        }
    }

    renderGraph() {
        if (!this.currentData || !this.currentData.nodes) return;

        const { nodes, edges } = this.currentData;

        // Update simulation with new data
        this.simulation.nodes(nodes);
        this.simulation.force('link').links(edges);

        // Render nodes
        this.renderNodes(nodes);
        
        // Render edges
        this.renderEdges(edges);
        
        // Update heatmap if enabled
        if (this.options.enableHeatmap) {
            this.updateHeatmap(nodes);
        }
        
        // Start/restart simulation
        this.simulation.alpha(1).restart();
    }

    renderNodes(nodes) {
        const nodeSelection = this.nodeLayer
            .selectAll('.performance-node')
            .data(nodes, d => d.id);

        // Remove old nodes
        nodeSelection.exit()
            .transition()
            .duration(500)
            .attr('opacity', 0)
            .remove();

        // Add new nodes
        const nodeEnter = nodeSelection.enter()
            .append('g')
            .attr('class', 'performance-node')
            .attr('opacity', 0)
            .call(this.createNodeDragBehavior());

        // Create node visual elements
        this.createNodeElements(nodeEnter);

        // Update all nodes
        const nodeUpdate = nodeEnter.merge(nodeSelection);
        
        nodeUpdate.transition()
            .duration(500)
            .attr('opacity', 1);

        // Update node properties based on performance data
        this.updateNodeAppearance(nodeUpdate);
        
        // Update labels
        this.updateNodeLabels(nodeUpdate);
    }

    createNodeElements(nodeEnter) {
        // Main circle for the node
        nodeEnter.append('circle')
            .attr('class', 'node-circle')
            .attr('r', 20);

        // Performance indicator ring
        nodeEnter.append('circle')
            .attr('class', 'performance-ring')
            .attr('r', 25)
            .attr('fill', 'none')
            .attr('stroke-width', 3);

        // Hotspot indicator (when applicable)
        nodeEnter.append('circle')
            .attr('class', 'hotspot-indicator')
            .attr('r', 8)
            .attr('cx', 15)
            .attr('cy', -15)
            .style('opacity', 0);

        // Node label
        nodeEnter.append('text')
            .attr('class', 'node-label')
            .attr('text-anchor', 'middle')
            .attr('dy', 5);

        // Performance metrics text
        nodeEnter.append('text')
            .attr('class', 'node-metrics')
            .attr('text-anchor', 'middle')
            .attr('dy', 35)
            .style('font-size', '10px')
            .style('opacity', 0.7);
    }

    updateNodeAppearance(nodeUpdate) {
        nodeUpdate.select('.node-circle')
            .transition()
            .duration(300)
            .attr('r', d => this.sizeScale(d.performance.total_queries || 0))
            .attr('fill', d => this.getNodeColor(d))
            .attr('stroke', d => this.getNodeStrokeColor(d))
            .attr('stroke-width', d => d.performance.hotspot_score > 70 ? 3 : 1);

        nodeUpdate.select('.performance-ring')
            .transition()
            .duration(300)
            .attr('r', d => this.sizeScale(d.performance.total_queries || 0) + 8)
            .attr('stroke', d => this.colorScales.latency(d.performance.average_latency || 0))
            .attr('stroke-dasharray', d => d.performance.error_rate > 0 ? '5,5' : 'none');

        // Show/hide hotspot indicator
        nodeUpdate.select('.hotspot-indicator')
            .transition()
            .duration(300)
            .style('opacity', d => d.performance.hotspot_score > 70 ? 1 : 0)
            .attr('fill', '#ff4444');
    }

    updateNodeLabels(nodeUpdate) {
        nodeUpdate.select('.node-label')
            .text(d => d.table_name || d.label);

        nodeUpdate.select('.node-metrics')
            .text(d => {
                const qps = d.performance.queries_per_second || 0;
                const lat = d.performance.average_latency || 0;
                return `${qps.toFixed(1)} QPS | ${lat.toFixed(0)}ms`;
            });
    }

    renderEdges(edges) {
        const edgeSelection = this.edgeLayer
            .selectAll('.performance-edge')
            .data(edges, d => d.id);

        // Remove old edges
        edgeSelection.exit()
            .transition()
            .duration(500)
            .attr('opacity', 0)
            .remove();

        // Add new edges
        const edgeEnter = edgeSelection.enter()
            .append('g')
            .attr('class', 'performance-edge')
            .attr('opacity', 0);

        // Create edge elements
        this.createEdgeElements(edgeEnter);

        // Update all edges
        const edgeUpdate = edgeEnter.merge(edgeSelection);
        
        edgeUpdate.transition()
            .duration(500)
            .attr('opacity', 1);

        // Update edge appearance based on performance data
        this.updateEdgeAppearance(edgeUpdate);
    }

    createEdgeElements(edgeEnter) {
        // Main edge line
        edgeEnter.append('line')
            .attr('class', 'edge-line')
            .attr('stroke', '#ccc')
            .attr('stroke-width', 2);

        // Performance flow indicator
        if (this.options.enableAnimations) {
            edgeEnter.append('line')
                .attr('class', 'flow-indicator')
                .attr('stroke-dasharray', '5,10')
                .attr('stroke', '#4CAF50')
                .attr('opacity', 0.7);
        }

        // Edge label for performance metrics
        edgeEnter.append('text')
            .attr('class', 'edge-label')
            .attr('text-anchor', 'middle')
            .style('font-size', '8px')
            .style('opacity', 0);
    }

    updateEdgeAppearance(edgeUpdate) {
        edgeUpdate.select('.edge-line')
            .transition()
            .duration(300)
            .attr('stroke-width', d => this.thicknessScale(d.performance.query_frequency || 1))
            .attr('stroke', d => this.getEdgeColor(d))
            .attr('opacity', d => d.performance.query_frequency > 0 ? 0.8 : 0.3);

        // Update flow indicators for animated edges
        if (this.options.enableAnimations) {
            edgeUpdate.select('.flow-indicator')
                .attr('stroke-width', d => Math.max(1, this.thicknessScale(d.performance.query_frequency || 1) - 1))
                .style('animation-duration', d => `${Math.max(1, 10 - (d.performance.query_frequency || 0) / 10)}s`);
        }

        // Show edge labels on hover
        edgeUpdate
            .on('mouseenter', function(event, d) {
                d3.select(this).select('.edge-label')
                    .style('opacity', 1)
                    .text(`${(d.performance.query_frequency || 0).toFixed(1)} queries/sec`);
            })
            .on('mouseleave', function(event, d) {
                d3.select(this).select('.edge-label')
                    .style('opacity', 0);
            });
    }

    updateHeatmap(nodes) {
        if (!this.options.enableHeatmap) return;

        // Create or update heatmap background
        const heatmapData = this.generateHeatmapData(nodes);
        
        const heatmapSelection = this.backgroundLayer
            .selectAll('.heatmap-cell')
            .data(heatmapData);

        heatmapSelection.exit().remove();

        const heatmapEnter = heatmapSelection.enter()
            .append('rect')
            .attr('class', 'heatmap-cell')
            .attr('opacity', 0);

        const heatmapUpdate = heatmapEnter.merge(heatmapSelection);

        heatmapUpdate
            .transition()
            .duration(1000)
            .attr('x', d => d.x)
            .attr('y', d => d.y)
            .attr('width', d => d.width)
            .attr('height', d => d.height)
            .attr('fill', d => this.colorScales.hotspot(d.intensity))
            .attr('opacity', d => d.intensity > 0 ? 0.3 : 0);
    }

    generateHeatmapData(nodes) {
        const cellSize = 50;
        const cols = Math.ceil(this.options.width / cellSize);
        const rows = Math.ceil(this.options.height / cellSize);
        const heatmapData = [];

        for (let row = 0; row < rows; row++) {
            for (let col = 0; col < cols; col++) {
                const x = col * cellSize;
                const y = row * cellSize;
                const cellCenterX = x + cellSize / 2;
                const cellCenterY = y + cellSize / 2;

                // Calculate intensity based on nearby nodes
                let intensity = 0;
                nodes.forEach(node => {
                    if (node.x && node.y) {
                        const distance = Math.sqrt(
                            Math.pow(cellCenterX - node.x, 2) + 
                            Math.pow(cellCenterY - node.y, 2)
                        );
                        const maxDistance = 100;
                        if (distance < maxDistance) {
                            const contribution = (node.performance.hotspot_score || 0) * 
                                                (1 - distance / maxDistance);
                            intensity += contribution;
                        }
                    }
                });

                heatmapData.push({
                    x, y,
                    width: cellSize,
                    height: cellSize,
                    intensity: Math.min(100, intensity)
                });
            }
        }

        return heatmapData;
    }

    animateDataChanges() {
        if (!this.previousData || !this.currentData) return;

        // Animate node changes
        this.animateNodeChanges();
        
        // Animate edge flow
        if (this.options.enableAnimations) {
            this.animateEdgeFlow();
        }
    }

    animateNodeChanges() {
        const currentNodes = new Map(this.currentData.nodes.map(n => [n.id, n]));
        const previousNodes = new Map(this.previousData.nodes.map(n => [n.id, n]));

        // Animate performance changes
        currentNodes.forEach((currentNode, id) => {
            const previousNode = previousNodes.get(id);
            if (previousNode) {
                const performanceChange = 
                    (currentNode.performance.queries_per_second || 0) - 
                    (previousNode.performance.queries_per_second || 0);

                if (Math.abs(performanceChange) > 1) {
                    this.showPerformanceChangeIndicator(currentNode, performanceChange);
                }
            }
        });
    }

    animateEdgeFlow() {
        this.edgeLayer.selectAll('.flow-indicator')
            .style('animation', d => {
                if (d.performance.query_frequency > 0) {
                    const speed = Math.max(1, 10 - (d.performance.query_frequency / 10));
                    return `edgeFlow ${speed}s linear infinite`;
                }
                return 'none';
            });
    }

    showPerformanceChangeIndicator(node, change) {
        const indicator = this.overlayLayer
            .append('text')
            .attr('class', 'performance-change-indicator')
            .attr('x', node.x)
            .attr('y', node.y - 30)
            .attr('text-anchor', 'middle')
            .style('font-size', '12px')
            .style('font-weight', 'bold')
            .style('fill', change > 0 ? '#4CAF50' : '#f44336')
            .text(`${change > 0 ? '+' : ''}${change.toFixed(1)} QPS`)
            .style('opacity', 0);

        indicator
            .transition()
            .duration(200)
            .style('opacity', 1)
            .transition()
            .delay(1500)
            .duration(500)
            .style('opacity', 0)
            .attr('y', node.y - 50)
            .remove();
    }

    updateNodePositions() {
        this.nodeLayer.selectAll('.performance-node')
            .attr('transform', d => `translate(${d.x},${d.y})`);

        this.edgeLayer.selectAll('.performance-edge')
            .each(function(d) {
                const edge = d3.select(this);
                edge.select('.edge-line')
                    .attr('x1', d.source.x)
                    .attr('y1', d.source.y)
                    .attr('x2', d.target.x)
                    .attr('y2', d.target.y);

                edge.select('.flow-indicator')
                    .attr('x1', d.source.x)
                    .attr('y1', d.source.y)
                    .attr('x2', d.target.x)
                    .attr('y2', d.target.y);

                edge.select('.edge-label')
                    .attr('x', (d.source.x + d.target.x) / 2)
                    .attr('y', (d.source.y + d.target.y) / 2);
            });
    }

    getNodeColor(node) {
        const latency = node.performance.average_latency || 0;
        if (latency > 500) return '#f44336';
        if (latency > 200) return '#ff9800';
        if (latency > 100) return '#ffeb3b';
        return '#4caf50';
    }

    getNodeStrokeColor(node) {
        if (node.performance.error_rate > 0) return '#f44336';
        if (node.performance.hotspot_score > 70) return '#ff9800';
        return '#ccc';
    }

    getEdgeColor(edge) {
        const frequency = edge.performance.query_frequency || 0;
        return this.colorScales.throughput(frequency);
    }

    createPerformanceLegend() {
        const legend = d3.select(`#${this.containerId}`)
            .append('div')
            .attr('class', 'performance-legend')
            .style('position', 'absolute')
            .style('top', '10px')
            .style('right', '10px')
            .style('background', 'rgba(255,255,255,0.9)')
            .style('padding', '10px')
            .style('border-radius', '5px')
            .style('font-size', '12px');

        legend.append('h4')
            .style('margin', '0 0 5px 0')
            .text('Performance Legend');

        const legendItems = [
            { color: '#4caf50', text: 'Good Performance (< 100ms)' },
            { color: '#ffeb3b', text: 'Fair Performance (100-200ms)' },
            { color: '#ff9800', text: 'Slow Performance (200-500ms)' },
            { color: '#f44336', text: 'Poor Performance (> 500ms)' }
        ];

        const legendList = legend.append('ul')
            .style('list-style', 'none')
            .style('padding', '0')
            .style('margin', '0');

        legendList.selectAll('li')
            .data(legendItems)
            .enter()
            .append('li')
            .style('margin', '2px 0')
            .each(function(d) {
                const li = d3.select(this);
                li.append('span')
                    .style('display', 'inline-block')
                    .style('width', '12px')
                    .style('height', '12px')
                    .style('background-color', d.color)
                    .style('margin-right', '5px')
                    .style('vertical-align', 'middle');
                li.append('span')
                    .text(d.text);
            });
    }

    createMetricsPanel() {
        this.metricsPanel = d3.select(`#${this.containerId}`)
            .append('div')
            .attr('class', 'metrics-panel')
            .style('position', 'absolute')
            .style('bottom', '10px')
            .style('left', '10px')
            .style('background', 'rgba(255,255,255,0.9)')
            .style('padding', '10px')
            .style('border-radius', '5px')
            .style('min-width', '200px')
            .style('font-size', '12px');

        this.metricsPanel.append('h4')
            .style('margin', '0 0 5px 0')
            .text('Real-time Metrics');

        this.metricsContent = this.metricsPanel.append('div');
    }

    updateMetricsPanel(metrics) {
        if (!this.metricsContent) return;

        const metricsData = [
            { label: 'Total QPS', value: metrics.database_metrics?.queries_per_second?.toFixed(1) || '0' },
            { label: 'Avg Latency', value: `${metrics.database_metrics?.average_latency?.toFixed(0) || '0'}ms` },
            { label: 'Active Connections', value: metrics.active_connections || '0' },
            { label: 'Slow Queries', value: metrics.database_metrics?.slow_queries || '0' }
        ];

        const metricsSelection = this.metricsContent
            .selectAll('.metric-item')
            .data(metricsData);

        const metricsEnter = metricsSelection.enter()
            .append('div')
            .attr('class', 'metric-item')
            .style('margin', '2px 0');

        metricsEnter.append('span')
            .attr('class', 'metric-label')
            .style('font-weight', 'bold');

        metricsEnter.append('span')
            .attr('class', 'metric-value')
            .style('float', 'right');

        const metricsUpdate = metricsEnter.merge(metricsSelection);
        
        metricsUpdate.select('.metric-label')
            .text(d => `${d.label}: `);
        
        metricsUpdate.select('.metric-value')
            .text(d => d.value);
    }

    createAlertPanel() {
        this.alertPanel = d3.select(`#${this.containerId}`)
            .append('div')
            .attr('class', 'alert-panel')
            .style('position', 'absolute')
            .style('top', '10px')
            .style('left', '10px')
            .style('max-width', '300px')
            .style('z-index', '1000');
    }

    showAlert(alert) {
        const alertElement = this.alertPanel
            .append('div')
            .attr('class', `alert alert-${alert.severity}`)
            .style('background', this.getAlertColor(alert.severity))
            .style('color', 'white')
            .style('padding', '10px')
            .style('margin', '5px 0')
            .style('border-radius', '5px')
            .style('opacity', '0');

        alertElement.append('strong')
            .text(alert.title);

        alertElement.append('div')
            .style('font-size', '12px')
            .text(alert.description);

        alertElement
            .transition()
            .duration(300)
            .style('opacity', '1');

        // Auto-remove after 10 seconds
        setTimeout(() => {
            alertElement
                .transition()
                .duration(300)
                .style('opacity', '0')
                .remove();
        }, 10000);
    }

    getAlertColor(severity) {
        switch (severity) {
            case 'critical': return '#f44336';
            case 'high': return '#ff9800';
            case 'medium': return '#ffeb3b';
            case 'low': return '#4caf50';
            default: return '#2196f3';
        }
    }

    createControlPanel() {
        const controls = d3.select(`#${this.containerId}`)
            .append('div')
            .attr('class', 'control-panel')
            .style('position', 'absolute')
            .style('top', '200px')
            .style('right', '10px')
            .style('background', 'rgba(255,255,255,0.9)')
            .style('padding', '10px')
            .style('border-radius', '5px');

        controls.append('h4')
            .style('margin', '0 0 10px 0')
            .text('Controls');

        // Animation toggle
        const animationControl = controls.append('label')
            .style('display', 'block')
            .style('margin', '5px 0');

        animationControl.append('input')
            .attr('type', 'checkbox')
            .attr('checked', this.options.enableAnimations)
            .on('change', (event) => {
                this.options.enableAnimations = event.target.checked;
            });

        animationControl.append('span')
            .text(' Enable Animations');

        // Heatmap toggle
        const heatmapControl = controls.append('label')
            .style('display', 'block')
            .style('margin', '5px 0');

        heatmapControl.append('input')
            .attr('type', 'checkbox')
            .attr('checked', this.options.enableHeatmap)
            .on('change', (event) => {
                this.options.enableHeatmap = event.target.checked;
                if (this.currentData) {
                    this.updateHeatmap(this.currentData.nodes);
                }
            });

        heatmapControl.append('span')
            .text(' Enable Heatmap');
    }

    showStatus(message, type = 'info') {
        const statusElement = d3.select('body')
            .append('div')
            .style('position', 'fixed')
            .style('top', '10px')
            .style('left', '50%')
            .style('transform', 'translateX(-50%)')
            .style('background', this.getStatusColor(type))
            .style('color', 'white')
            .style('padding', '10px 20px')
            .style('border-radius', '5px')
            .style('z-index', '10000')
            .style('opacity', '0')
            .text(message);

        statusElement
            .transition()
            .duration(300)
            .style('opacity', '1');

        setTimeout(() => {
            statusElement
                .transition()
                .duration(300)
                .style('opacity', '0')
                .remove();
        }, 3000);
    }

    getStatusColor(type) {
        switch (type) {
            case 'success': return '#4caf50';
            case 'warning': return '#ff9800';
            case 'error': return '#f44336';
            default: return '#2196f3';
        }
    }

    createNodeDragBehavior() {
        return d3.drag()
            .on('start', (event, d) => {
                if (!event.active) this.simulation.alphaTarget(0.3).restart();
                d.fx = d.x;
                d.fy = d.y;
            })
            .on('drag', (event, d) => {
                d.fx = event.x;
                d.fy = event.y;
            })
            .on('end', (event, d) => {
                if (!event.active) this.simulation.alphaTarget(0);
                d.fx = null;
                d.fy = null;
            });
    }

    setupEventHandlers() {
        // Handle window resize
        window.addEventListener('resize', () => {
            this.options.width = this.container.clientWidth;
            this.options.height = this.container.clientHeight;
            this.svg.attr('width', this.options.width)
                   .attr('height', this.options.height);
            this.simulation.force('center', 
                d3.forceCenter(this.options.width / 2, this.options.height / 2));
        });
    }

    destroy() {
        if (this.websocket) {
            this.websocket.close();
        }
        
        if (this.animationFrameId) {
            cancelAnimationFrame(this.animationFrameId);
        }
        
        this.simulation.stop();
        this.container.innerHTML = '';
    }
}

// CSS styles for animations (to be added to CSS file)
const performanceStyles = `
@keyframes edgeFlow {
    0% { stroke-dashoffset: 0; }
    100% { stroke-dashoffset: 15; }
}

.performance-node {
    cursor: pointer;
}

.performance-node:hover {
    filter: brightness(1.2);
}

.performance-edge:hover {
    opacity: 1 !important;
}

.alert-critical {
    animation: pulse 1s infinite;
}

@keyframes pulse {
    0% { opacity: 0.8; }
    50% { opacity: 1; }
    100% { opacity: 0.8; }
}

.performance-legend, .metrics-panel, .control-panel {
    box-shadow: 0 2px 10px rgba(0,0,0,0.1);
}
`;

// Add styles to document
const styleSheet = document.createElement('style');
styleSheet.textContent = performanceStyles;
document.head.appendChild(styleSheet);

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = PerformanceGraphVisualizer;
}
