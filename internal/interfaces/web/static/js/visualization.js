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
        this.initialize();
    }

    async initialize() {
        console.log('Initializing visualization...');
        
        try {
            // Load graph data from REST API instead of direct Neo4j connection
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

            // Process nodes from API
            if (graphData.nodes) {
                graphData.nodes.forEach(node => {
                    const displayLabel = node.properties.name || 
                                       node.properties.nazev || 
                                       node.properties.php_code || 
                                       node.properties.id || 
                                       node.label || 
                                       'N/A';
                    
                    const nodeColor = node.label === 'NodePHPAction' ? '#97C2FC' : 
                                    node.label === 'PHPAction' ? '#FFA500' : '#97C2FC';
                    
                    nodes.add({
                        id: node.id,
                        label: displayLabel,
                        title: Object.entries(node.properties)
                            .map(([key, value]) => `${key}: ${value}`)
                            .join('\n') + '\nType: ' + node.label,
                        color: nodeColor,
                        group: node.label
                    });
                });
            }

            // Process relationships from API
            if (graphData.relationships) {
                graphData.relationships.forEach(rel => {
                    edges.add({
                        from: rel.from,
                        to: rel.to,
                        label: rel.type,
                        arrows: 'to',
                        title: `Type: ${rel.type}\nProperties: ${JSON.stringify(rel.properties)}`
                    });
                });
            }

            console.log('Processed nodes:', nodes.get().length);
            console.log('Processed edges:', edges.get().length);
            console.log('Nodes:', nodes.get());
            console.log('Edges:', edges.get());

            const options = {
                nodes: {
                    shape: 'dot',
                    size: 25,
                    font: {
                        size: 14,
                        face: 'Arial',
                        vadjust: 3,
                        background: 'white',
                        strokeWidth: 0,
                        color: '#000000'
                    },
                    borderWidth: 2
                },
                edges: {
                    arrows: { to: true },
                    color: '#848484',
                    font: {
                        size: 12,
                        align: 'middle',
                        background: 'white'
                    },
                    width: 1,
                    smooth: {
                        type: 'continuous'
                    }
                },
                physics: {
                    enabled: true,
                    solver: 'forceAtlas2Based',
                    forceAtlas2Based: {
                        gravitationalConstant: -50,
                        centralGravity: 0.01,
                        springLength: 100,
                        springConstant: 0.08
                    },
                    stabilization: {
                        enabled: true,
                        iterations: 1000,
                        updateInterval: 25
                    }
                },
                groups: {
                    NodePHPAction: {
                        color: { background: '#97C2FC', border: '#2B7CE9' }
                    },
                    PHPAction: {
                        color: { background: '#FFA500', border: '#FF8C00' }
                    }
                }
            };

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

        } catch (error) {
            console.error('Error initializing visualization:', error);
        }
    }

    initializeEventListeners() {
        if (!this.network) return;

        this.network.on('click', (params) => {
            if (params.nodes.length > 0) {
                console.log('Node clicked:', params.nodes[0]);
            }
        });

        this.network.on('doubleClick', (params) => {
            if (params.nodes.length > 0) {
                console.log('Node double-clicked:', params.nodes[0]);
            }
        });
    }
}

window.addEventListener('load', () => {
    console.log('Page loaded, starting visualization...');
    new GraphVisualizer();
});
