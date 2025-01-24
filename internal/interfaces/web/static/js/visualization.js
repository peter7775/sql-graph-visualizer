/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

class GraphVisualizer {
    constructor() {
        this.viz = null;
        this.data = null;
        this.network = null;
        this.currentLayout = 'force';
        this.filters = {
            nodes: new Set(),
            relations: new Set()
        };
        
        this.initializeEventListeners();
    }

    async initialize() {
        try {
            console.log('Načítám data z API...');
            const response = await fetch('/api/graph');
            if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
            }
            this.data = await response.json();
            console.log('Načtená data:', this.data);
        
            await this.createVisualization();
            } catch (error) {
                console.error('Chyba při načítání dat:', error);
            }
    }



    async createVisualization() {
        const container = document.getElementById('viz');
        
        // Připravíme data pro vis.js
        console.log('Zpracovávám uzly:', this.data.nodes);
        const nodes = new vis.DataSet(this.data.nodes.map(node => {
            // Dekódování base64 hodnot
            const properties = {};
            for (let key in node.properties) {
                try {
                    properties[key] = atob(node.properties[key]);
                } catch (e) {
                    properties[key] = node.properties[key];
                }
            }
            
            const label = properties.name || node.id;
            console.log('Vytvářím uzel:', { id: node.id, label, properties });
            return {
                id: node.id,
                label: label,
                title: JSON.stringify(properties, null, 2),
                group: node.label,
                color: node.label === 'Person' ? '#97C2FC' : '#FB7E81'
            };
        }));

        console.log('Zpracovávám vztahy:', this.data.relationships);
        const edges = new vis.DataSet(this.data.relationships.map(rel => {
            console.log('Vytvářím vztah:', rel);
            return {
                from: rel.from,
                to: rel.to,
                label: rel.type || 'VZTAH',
                title: JSON.stringify(rel.properties, null, 2),
                arrows: 'to'
            };
        }));

        const options = {
            nodes: {
                shape: 'dot',
                size: 30,
                font: {
                    size: 14,
                    face: 'Tahoma'
                },
                borderWidth: 2,
                shadow: true
            },
            edges: {
                width: 2,
                font: {
                    size: 14,
                    face: 'Tahoma'
                },
                arrows: {
                    to: {
                        enabled: true,
                        scaleFactor: 1
                    }
                },
                shadow: true
            },
            physics: {
                enabled: true,
                solver: 'forceAtlas2Based',
                forceAtlas2Based: {
                    gravitationalConstant: -26,
                    centralGravity: 0.005,
                    springLength: 230,
                    springConstant: 0.18
                },
                stabilization: {
                    enabled: true,
                    iterations: 1000,
                    updateInterval: 25
                }
            },
            interaction: {
                hover: true,
                tooltipDelay: 200,
                hideEdgesOnDrag: true,
                navigationButtons: true,
                keyboard: true
            }
        };

        console.log('Vytvářím síť s daty:', { nodes: nodes.get(), edges: edges.get() });
        this.network = new vis.Network(container, { nodes, edges }, options);

        this.network.on('stabilizationProgress', function(params) {
            console.log('Stabilizace:', Math.round(params.iterations/params.total * 100), '%');
        });

        this.network.on('stabilizationIterationsDone', function() {
            console.log('Stabilizace dokončena');
        });
    }

    initializeEventListeners() {
        // Vyhledávání
        const searchInput = document.getElementById('search');
        searchInput.addEventListener('input', _.debounce((e) => {
            const term = e.target.value.toLowerCase();
            if (term.length < 2) {
                const container = document.getElementById('searchResults');
                container.classList.add('d-none');
                return;
            }
            
            const results = this.data.nodes.filter(node => {
                const properties = Object.values(node.properties || {}).map(v => String(v).toLowerCase());
                return node.label.toLowerCase().includes(term) || 
                       properties.some(v => v.includes(term));
            });
            this.showSearchResults(results);
        }, 300));

        // Změna layoutu
        document.getElementById('layout').addEventListener('change', (e) => {
            this.currentLayout = e.target.value;
            this.updateLayout();
        });

        // Vyčištění
        document.getElementById('clear').addEventListener('click', () => {
            if (this.network) {
                this.network.fit();
            }
        });
    }

    showSearchResults(results) {
        const container = document.getElementById('searchResults');
        container.innerHTML = '';
        container.classList.remove('d-none');

        if (results.length === 0) {
            const div = document.createElement('div');
            div.className = 'p-2 text-muted';
            div.textContent = 'Žádné výsledky';
            container.appendChild(div);
            return;
        }

        results.forEach(result => {
            const div = document.createElement('div');
            div.className = 'p-2 border-bottom';
            const name = result.properties.name || result.id;
            div.textContent = `${result.label}: ${name}`;
            div.addEventListener('click', () => {
                if (this.network) {
                    this.network.focus(result.id, {
                        scale: 1.5,
                        animation: {
                            duration: 1000,
                            easingFunction: 'easeInOutQuad'
                        }
                    });
                    this.network.selectNodes([result.id]);
                }
                container.classList.add('d-none');
            });
            container.appendChild(div);
        });
    }

    updateLayout() {
        if (!this.network) return;

        const options = {
            physics: {
                enabled: true,
                stabilization: {
                    enabled: true,
                    iterations: 1000,
                    updateInterval: 25
                }
            }
        };

        switch(this.currentLayout) {
            case 'hierarchical':
                options.layout = {
                    hierarchical: {
                        direction: 'UD',
                        sortMethod: 'directed',
                        nodeSpacing: 150,
                        treeSpacing: 200
                    }
                };
                break;
            case 'circular':
                options.layout = {
                    improvedLayout: true,
                    randomSeed: 2
                };
                options.physics = {
                    enabled: true,
                    solver: 'forceAtlas2Based',
                    forceAtlas2Based: {
                        gravitationalConstant: -50,
                        centralGravity: 0.01,
                        springLength: 200,
                        springConstant: 0.08
                    }
                };
                break;
            default:
                options.layout = {
                    improvedLayout: true,
                    randomSeed: 1
                };
                options.physics = {
                    enabled: true,
                    solver: 'forceAtlas2Based',
                    forceAtlas2Based: {
                        gravitationalConstant: -26,
                        centralGravity: 0.005,
                        springLength: 230,
                        springConstant: 0.18
                    }
                };
        }

        this.network.setOptions(options);
    }
}

// Inicializace při načtení stránky
window.addEventListener('load', () => {
    console.log('Stránka načtena, inicializuji vizualizaci...');
    const visualizer = new GraphVisualizer();
    visualizer.initialize();
}); 