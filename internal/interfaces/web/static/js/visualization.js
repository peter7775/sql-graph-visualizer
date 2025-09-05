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
                });
            }

            if (graphData.relationships) {
                graphData.relationships.forEach(rel => {
                    let edgeColor = '#848484';
                    let edgeWidth = 2;
                    let edgeLabel = rel.type;
                    
                    switch(rel.type) {
                        case 'LEADS':
                            edgeColor = '#D0021B';
                            edgeWidth = 3;
                            break;
                        case 'MEMBER_OF':
                            edgeColor = '#7ED321';
                            break;
                        case 'HAS_SKILL':
                            edgeColor = '#50E3C2';
                            break;
                        case 'REQUIRES_SKILL':
                            edgeColor = '#F5A623';
                            break;
                        case 'ASSIGNED_TO':
                            edgeColor = '#BD10E0';
                            break;
                        case 'DEPENDS_ON':
                            edgeColor = '#FF6B6B';
                            edgeWidth = 3;
                            break;
                        case 'SUBTASK_OF':
                            edgeColor = '#9013FE';
                            break;
                        case 'SKILL_COMPATIBLE':
                            edgeColor = '#4A90E2';
                            edgeWidth = 1;
                            break;
                        case 'EXPERT_IN':
                            edgeColor = '#9013FE';
                            edgeWidth = 4;
                            break;
                        case 'ENHANCED_VIEW_OF':
                            edgeColor = '#FF9500';
                            edgeWidth = 3;
                            break;
                    }
                    
                    let edgeTooltip = `Relationship: ${rel.type}`;
                    if (rel.properties && Object.keys(rel.properties).length > 0) {
                        edgeTooltip += '\nProperties:\n' + 
                            Object.entries(rel.properties)
                                .map(([key, value]) => `  ${key}: ${value}`)
                                .join('\n');
                    }
                    
                    edges.add({
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
                });
            }

            console.log('Processed nodes:', nodes.get().length);
            console.log('Processed edges:', edges.get().length);
            console.log('Nodes:', nodes.get());
            console.log('Edges:', edges.get());

                const options = {
                nodes: {
                    shape: 'dot',
                    size: 30,
                    font: {
                        size: 12,
                        face: 'Arial',
                        vadjust: 0,
                        background: 'rgba(255,255,255,0.8)',
                        strokeWidth: 1,
                        color: '#000000'
                    },
                    borderWidth: 2,
                    chosen: {
                        node: function(values, id, selected, hovering) {
                            values.size = 35;
                            values.borderWidth = 3;
                        }
                    }
                },
                edges: {
                    arrows: { to: { enabled: true, scaleFactor: 0.8 } },
                    color: {
                        color: '#848484',
                        highlight: '#FF6B6B',
                        hover: '#FF6B6B'
                    },
                    font: {
                        size: 10,
                        align: 'middle',
                        background: 'rgba(255,255,255,0.9)',
                        strokeWidth: 2,
                        strokeColor: '#ffffff'
                    },
                    width: 2,
                    smooth: {
                        type: 'dynamic',
                        roundness: 0.2
                    },
                    chosen: {
                        edge: function(values, id, selected, hovering) {
                            values.width = 4;
                        }
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
                        iterations: 1500,
                        updateInterval: 50
                    },
                    maxVelocity: 30,
                    minVelocity: 0.1,
                    timestep: 0.35
                },
                groups: {
                    User: {
                        color: { background: '#4A90E2', border: '#2E5C8A' },
                        shape: 'dot',
                        size: 25
                    },
                    Team: {
                        color: { background: '#7ED321', border: '#5BA91A' },
                        shape: 'diamond',
                        size: 30
                    },
                    Project: {
                        color: { background: '#F5A623', border: '#D18A00' },
                        shape: 'box',
                        size: 35
                    },
                    Task: {
                        color: { background: '#BD10E0', border: '#9013FE' },
                        shape: 'dot',
                        size: 20
                    },
                    Skill: {
                        color: { background: '#50E3C2', border: '#00D4AA' },
                        shape: 'triangle',
                        size: 25
                    },
                    HighImpactProject: {
                        color: { background: '#D0021B', border: '#B71C1C' },
                        shape: 'star',
                        size: 40
                    },
                    SkillExpert: {
                        color: { background: '#9013FE', border: '#6A1B9A' },
                        shape: 'triangleDown',
                        size: 30
                    },
                    TeamSummary: {
                        color: { background: '#FFC107', border: '#FF8F00' },
                        shape: 'hexagon',
                        size: 35
                    },
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

        this.initializeControlButtons();
        this.initializeSearch();
        this.initializeLayoutSelector();
    }

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
}

window.addEventListener('load', () => {
    console.log('Page loaded, starting visualization...');
    new GraphVisualizer();
});
