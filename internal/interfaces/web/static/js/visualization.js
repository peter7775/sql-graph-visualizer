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
        console.log('Inicializace vizualizace...');
        
        try {
            const response = await fetch('/config');
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            const serverConfig = await response.json();
            console.log('Načtena konfigurace ze serveru:', serverConfig);

            const container = document.getElementById('viz');
            if (!container) {
                throw new Error('Container "viz" nebyl nalezen!');
            }
            console.log('Container nalezen:', container);

            // Připojení k Neo4j a načtení dat
            const driver = neo4j.driver(
                serverConfig.neo4j.uri,
                neo4j.auth.basic(serverConfig.neo4j.username, serverConfig.neo4j.password)
            );
            
            const session = driver.session();
            try {
                const result = await session.run(`
                    MATCH (n)
                    WITH n LIMIT 100
                    OPTIONAL MATCH (n)-[r]-(m)
                    RETURN n, r, m
                `);
                console.log('Data z Neo4j:', result);

                // Převod dat do formátu pro vis.js
                const nodes = new vis.DataSet();
                const edges = new vis.DataSet();
                const processedNodes = new Set();

                result.records.forEach(record => {
                    const sourceNode = record.get('n');
                    const relationship = record.get('r');
                    const targetNode = record.get('m');

                    if (sourceNode && !processedNodes.has(sourceNode.identity.toString())) {
                        processedNodes.add(sourceNode.identity.toString());
                        nodes.add({
                            id: sourceNode.identity.toString(),
                            label: sourceNode.properties.name || 
                                  sourceNode.properties.nazev || 
                                  sourceNode.properties.php_code || 
                                  sourceNode.properties.id || 
                                  'N/A',
                            title: Object.entries(sourceNode.properties)
                                .map(([key, value]) => `${key}: ${value}`)
                                .join('\n'),
                            color: '#97C2FC'
                        });
                    }

                    if (targetNode && !processedNodes.has(targetNode.identity.toString())) {
                        processedNodes.add(targetNode.identity.toString());
                        nodes.add({
                            id: targetNode.identity.toString(),
                            label: targetNode.properties.name || 
                                  targetNode.properties.nazev || 
                                  targetNode.properties.php_code || 
                                  targetNode.properties.id || 
                                  'N/A',
                            title: Object.entries(targetNode.properties)
                                .map(([key, value]) => `${key}: ${value}`)
                                .join('\n'),
                            color: '#97C2FC'
                        });
                    }

                    if (relationship) {
                        edges.add({
                            from: sourceNode.identity.toString(),
                            to: targetNode.identity.toString(),
                            label: relationship.type,
                            arrows: 'to'
                        });
                    }
                });

                console.log('Nodes:', nodes.get());
                console.log('Edges:', edges.get());

                // Konfigurace sítě
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
                    }
                };

                // Vytvoření sítě
                const data = { nodes, edges };
                this.network = new vis.Network(container, data, options);
                console.log('Síť vytvořena:', this.network);

                // Event listeners
                this.network.on('stabilizationProgress', function(params) {
                    console.log('Stabilizace:', Math.round(params.iterations/params.total * 100), '%');
                });

                this.network.on('stabilizationIterationsDone', function() {
                    console.log('Stabilizace dokončena');
                });

                this.initializeEventListeners();

            } catch (error) {
                console.error('Chyba při načítání dat z Neo4j:', error);
            } finally {
                await session.close();
                await driver.close();
            }

        } catch (error) {
            console.error('Chyba při inicializaci vizualizace:', error);
        }
    }

    initializeEventListeners() {
        if (!this.network) return;

        this.network.on('click', (params) => {
            if (params.nodes.length > 0) {
                console.log('Kliknutí na uzel:', params.nodes[0]);
            }
        });

        this.network.on('doubleClick', (params) => {
            if (params.nodes.length > 0) {
                console.log('Dvojklik na uzel:', params.nodes[0]);
            }
        });
    }
}

// Inicializace po načtení stránky
window.addEventListener('load', () => {
    console.log('Stránka načtena, spouštím vizualizaci...');
    new GraphVisualizer();
}); 