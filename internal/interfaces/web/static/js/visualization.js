class GraphVisualizer {
    constructor() {
        this.viz = null;
        this.config = null;
        this.currentLayout = 'force';
        this.filters = {
            nodes: new Set(),
            relations: new Set()
        };
        
        this.initializeEventListeners();
    }

    async initialize() {
        const response = await fetch('/api/visualization/config');
        this.config = await response.json();
        
        await this.createVisualization();
    }

    async createVisualization() {
        const vizConfig = {
            container_id: "viz",
            server_url: this.config.neo4j.uri,
            server_user: this.config.neo4j.username,
            server_password: this.config.neo4j.password,
            labels: {
                "User": {
                    caption: "name",
                    size: "pagerank",
                    community: "community",
                    title_properties: ["name", "email", "role"]
                },
                "Department": {
                    caption: "name",
                    size: "pagerank",
                    community: "community",
                    title_properties: ["name", "description"]
                }
            },
            relationships: {
                "MANAGES": {
                    thickness: "weight",
                    caption: false
                },
                "BELONGS_TO": {
                    thickness: "weight",
                    caption: false
                }
            },
            initial_cypher: this.buildQuery()
        };

        this.viz = new NeoVis.default(vizConfig);
        this.viz.render();
    }

    buildQuery() {
        let query = 'MATCH (n)';
        
        if (this.filters.nodes.size > 0) {
            query += ` WHERE n:${Array.from(this.filters.nodes).join(' OR n:')}`;
        }
        
        query += ' MATCH (n)-[r]->(m)';
        
        if (this.filters.relations.size > 0) {
            query += ` WHERE type(r) IN ['${Array.from(this.filters.relations).join("','")}']`;
        }
        
        return query + ' RETURN n, r, m LIMIT 100';
    }

    initializeEventListeners() {
        // Vyhledávání
        const searchInput = document.getElementById('search');
        searchInput.addEventListener('input', _.debounce(async (e) => {
            const term = e.target.value;
            if (term.length < 2) return;
            
            const response = await fetch(`/api/visualization/search?term=${encodeURIComponent(term)}`);
            const results = await response.json();
            this.showSearchResults(results);
        }, 300));

        // Změna layoutu
        document.getElementById('layout').addEventListener('change', (e) => {
            this.currentLayout = e.target.value;
            this.updateLayout();
        });

        // Filtry
        document.getElementById('filterNodes').addEventListener('click', () => {
            this.showNodeFilterModal();
        });

        document.getElementById('filterRelations').addEventListener('click', () => {
            this.showRelationFilterModal();
        });

        // Export
        document.getElementById('export').addEventListener('click', () => {
            this.exportGraph();
        });

        // Vyčištění
        document.getElementById('clear').addEventListener('click', () => {
            this.clearFilters();
        });
    }

    async showSearchResults(results) {
        const container = document.getElementById('searchResults');
        container.innerHTML = '';
        container.classList.remove('d-none');

        results.forEach(result => {
            const div = document.createElement('div');
            div.className = 'p-2 border-bottom';
            div.textContent = result.name;
            div.addEventListener('click', () => {
                this.focusNode(result.id);
            });
            container.appendChild(div);
        });
    }

    updateLayout() {
        // Implementace různých layoutů
        switch(this.currentLayout) {
            case 'hierarchical':
                this.viz.updateWithOptions({ layout: { hierarchical: true } });
                break;
            case 'circular':
                this.viz.updateWithOptions({ layout: { circular: true } });
                break;
            default:
                this.viz.updateWithOptions({ layout: { randomSeed: 1 } });
        }
    }

    async exportGraph() {
        const format = await this.showExportDialog();
        const response = await fetch('/api/visualization/export', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ format })
        });

        if (format === 'image') {
            const blob = await response.blob();
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'graph.png';
            a.click();
        } else {
            const data = await response.json();
            const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'graph.json';
            a.click();
        }
    }

    showExportDialog() {
        return new Promise(resolve => {
            // Implementace dialogu pro výběr formátu exportu
        });
    }

    clearFilters() {
        this.filters.nodes.clear();
        this.filters.relations.clear();
        this.createVisualization();
    }
}

// Inicializace při načtení stránky
window.addEventListener('load', () => {
    const visualizer = new GraphVisualizer();
    visualizer.initialize();
}); 