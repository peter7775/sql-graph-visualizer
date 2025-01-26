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

    async fetchConfig() {
        try {
            const response = await fetch("http://localhost:8080/config");
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            const contentType = response.headers.get("content-type");
            if (!contentType || !contentType.includes("application/json")) {
                throw new TypeError("Odpověď není JSON");
            }
            const config = await response.json();
            return config;
        } catch (error) {
            console.error('Chyba při načítání konfigurace:', error);
            return null;
        }
    }

    async initialize() {
        try {
            console.log('Načítám konfiguraci...');
            const config = await this.fetchConfig();
            if (!config) {
                throw new Error('Konfigurace nebyla načtena.');
            }

            console.log('Načítám data z GraphQL...');
            const query = `
                query GetNodes {
                    nodes {
                        id
                        label
                        properties {
                            key
                            value
                        }
                    }
                }
            `;
            const response = await fetch('http://localhost:8080/graphql', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Accept': 'application/json',
                },
                body: JSON.stringify({ 
                    query: query,
                    operationName: 'GetNodes'
                }),
            });
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            const result = await response.json();
            console.log('GraphQL response:', result);
            this.data = result.data.nodes;
            console.log('Načtená data:', this.data);

            this.createVisualization(config);
        } catch (error) {
            console.error('Chyba při inicializaci:', error);
        }
    }

    createVisualization(config) {
        console.log('Konfigurace Neo4j:', config);

        const vizConfig = {
            container_id: "viz",
            server_url: config.Neo4j.URI,
            server_user: config.Neo4j.User,
            server_password: config.Neo4j.Password,
            labels: {
                "Node": {
                    "caption": "label",
                    "size": "pagerank",
                    "community": "community"
                }
            },
            relationships: {
                "RELATIONSHIP": {
                    "thickness": "weight",
                    "caption": false
                }
            },
            initial_cypher: "MATCH (n)-[r]->(m) RETURN n,r,m"
        };

        console.log('Konfigurace vizualizace:', vizConfig);

        this.viz = new NeoVis.default(vizConfig);
        this.viz.render();
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