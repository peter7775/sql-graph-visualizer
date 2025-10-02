package deployment

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"sql-graph-visualizer/internal/application/ports"
	"sql-graph-visualizer/internal/domain/aggregates/graph"
	"sql-graph-visualizer/internal/domain/models"
)


type RailwayDeployment struct {
	logger *logrus.Logger
}


func NewRailwayDeployment(logger *logrus.Logger) ports.DeploymentPort {
	return &RailwayDeployment{
		logger: logger,
	}
}


func (r *RailwayDeployment) GetAPIPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // fallback for local development
	}
	r.logger.Infof("Railway: Using API port %s", port)
	return port
}


func (r *RailwayDeployment) GetVisualizationPort() string {
	if r.isRailwayEnvironment() {

		port := r.GetAPIPort()
		r.logger.Infof("Railway: Using visualization port %s (same as API for single port deployment)", port)
		return port
	}

	return "3000"
}


func (r *RailwayDeployment) ShouldStartVisualizationServer() bool {
	if r.isRailwayEnvironment() {
	
		r.logger.Info("Railway: Visualization integrated into API server (single port deployment)")
		return false
	}

	r.logger.Info("Local: Starting separate visualization server")
	return true
}


func (r *RailwayDeployment) GetEnvironmentInfo() map[string]interface{} {
	return map[string]interface{}{
		"platform":            r.GetPlatformName(),
		"railway_environment": r.getEnvOrDefault("RAILWAY_ENVIRONMENT", "not_set"),
		"railway_service_id":  r.getEnvOrDefault("RAILWAY_SERVICE_ID", "not_set"),
		"railway_project_id":  r.getEnvOrDefault("RAILWAY_PROJECT_ID", "not_set"),
		"port":                r.getEnvOrDefault("PORT", "not_set"),
		"force_full_mode":     r.getEnvOrDefault("FORCE_FULL_MODE", "not_set"),
	}
}


func (r *RailwayDeployment) ConfigureServer(server *http.Server) *http.Server {
	if r.isRailwayEnvironment() {
		r.logger.Info("Railway: Applying Railway-specific server configuration")
		
	
		server.ReadTimeout = 30 * time.Second
		server.ReadHeaderTimeout = 10 * time.Second
		server.WriteTimeout = 30 * time.Second
		server.IdleTimeout = 120 * time.Second
		
	
		if server.Addr == "" || server.Addr[0] != ':' {
			server.Addr = ":" + r.GetAPIPort()
		}
		
		r.logger.Infof("Railway: Server configured for Railway deployment on %s", server.Addr)
	} else {
		r.logger.Info("Railway: Local development mode - using default configuration")
	}
	
	return server
}


func (r *RailwayDeployment) GetPlatformName() string {
	if r.isRailwayEnvironment() {
		return "Railway"
	}
	return "Local"
}


func (r *RailwayDeployment) isRailwayEnvironment() bool {
	return os.Getenv("RAILWAY_ENVIRONMENT") != ""
}


func (r *RailwayDeployment) getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (r *RailwayDeployment) GetRailwayDatabaseConfig() map[string]string {
	config := make(map[string]string)
	
	if r.isRailwayEnvironment() {
		r.logger.Info("Railway: Loading Railway database configuration")
		
	
		config["mysql_host"] = r.getEnvOrDefault("MYSQL_HOST", "")
		config["mysql_port"] = r.getEnvOrDefault("MYSQL_PORT", "3306")
		config["mysql_user"] = r.getEnvOrDefault("MYSQL_USER", "")
		config["mysql_password"] = r.getEnvOrDefault("MYSQL_PASSWORD", "")
		config["mysql_database"] = r.getEnvOrDefault("MYSQL_DATABASE", "")
		
	
		config["neo4j_uri"] = r.getEnvOrDefault("NEO4J_URI", "")
		config["neo4j_user"] = r.getEnvOrDefault("NEO4J_USER", "neo4j")
		config["neo4j_password"] = r.getEnvOrDefault("NEO4J_PASSWORD", "")
		
		r.logger.Infof("Railway: Database config loaded - MySQL host: %s, Neo4j URI set: %t", 
			config["mysql_host"], config["neo4j_uri"] != "")
	}
	
	return config
}


func (r *RailwayDeployment) IsProductionMode() bool {
	railwayEnv := os.Getenv("RAILWAY_ENVIRONMENT")
	return railwayEnv == "production"
}


func (r *RailwayDeployment) GetMemoryLimit() int64 {
	if !r.isRailwayEnvironment() {
		return 0
	}
	

	if memStr := os.Getenv("RAILWAY_MEMORY_LIMIT"); memStr != "" {
		if mem, err := strconv.ParseInt(memStr, 10, 64); err == nil {
			return mem
		}
	}
	

	return 512
}

func (r *RailwayDeployment) GetHealthCheckConfig() map[string]interface{} {
	return map[string]interface{}{
		"enabled":     true,
		"path":        "/api/health",
		"port":        r.GetAPIPort(),
		"timeout":     30,
		"interval":    10,
		"retries":     3,
		"start_period": 60,
	}
}


func (r *RailwayDeployment) RegisterVisualizationRoutes(router *mux.Router, neo4jRepo ports.Neo4jPort, cfg *models.Config) error {
	if !r.isRailwayEnvironment() {
		r.logger.Info("Railway: Not in Railway environment - skipping visualization route registration")
		return nil
	}

	r.logger.Info("Railway: Registering visualization routes in API server for single-port deployment")


	router.HandleFunc("/api/graph", func(w http.ResponseWriter, req *http.Request) {
		r.logger.Info("Railway: Request to API endpoint /api/graph")

		graphInterface, err := neo4jRepo.ExportGraph("MATCH (n)-[r]->(m) RETURN n, r, m")
		if err != nil {
			r.logger.Errorf("Railway: Error retrieving graph data: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		g, ok := graphInterface.(*graph.GraphAggregate)
		if !ok {
			r.logger.Warn("Railway: Invalid graph type")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		response := struct {
			Nodes         []map[string]interface{} `json:"nodes"`
			Relationships []map[string]interface{} `json:"relationships"`
		}{
			Nodes:         make([]map[string]interface{}, 0),
			Relationships: make([]map[string]interface{}, 0),
		}

		for _, node := range g.GetNodes() {
			nodeData := map[string]interface{}{
				"id":         node.ID,
				"label":      node.Type,
				"properties": node.Properties,
			}
			response.Nodes = append(response.Nodes, nodeData)
		}

		for _, rel := range g.GetRelationships() {
			relData := map[string]interface{}{
				"from":       rel.SourceNode.ID,
				"to":         rel.TargetNode.ID,
				"type":       rel.Type,
				"properties": rel.Properties,
			}
			response.Relationships = append(response.Relationships, relData)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		r.logger.Infof("Railway: Sending graph response: %d nodes, %d relationships", len(response.Nodes), len(response.Relationships))

		if err := json.NewEncoder(w).Encode(response); err != nil {
			r.logger.Errorf("Railway: Error serializing graph response: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}).Methods("GET")

	
	router.HandleFunc("/config", func(w http.ResponseWriter, req *http.Request) {
		r.logger.Info("Railway: Request to /config endpoint")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		configResponse := map[string]interface{}{
			"neo4j": map[string]string{
				"uri":      cfg.Neo4j.URI,
				"username": cfg.Neo4j.User,
				"password": cfg.Neo4j.Password,
			},
		}

		if err := json.NewEncoder(w).Encode(configResponse); err != nil {
			r.logger.Errorf("Railway: Error encoding config response: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}).Methods("GET")

	
	webRoot := r.findProjectRoot()
	if webRoot != "" {
		staticPath := filepath.Join(webRoot, "internal", "interfaces", "web", "static")
		if _, err := os.Stat(staticPath); err == nil {
			r.logger.Infof("Railway: Serving static files from %s", staticPath)
			fs := http.FileServer(http.Dir(staticPath))
			router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
		} else {
			r.logger.Warnf("Railway: Static files directory not found: %s", staticPath)
		}


		visualizationPath := filepath.Join(webRoot, "internal", "interfaces", "web", "templates", "visualization.html")
		if _, err := os.Stat(visualizationPath); err == nil {
			r.logger.Infof("Railway: Registering visualization page at /visualization")
			router.HandleFunc("/visualization", func(w http.ResponseWriter, req *http.Request) {
				r.logger.Info("Railway: Serving visualization page")
				http.ServeFile(w, req, visualizationPath)
			}).Methods("GET")
		} else {
			r.logger.Warnf("Railway: Visualization template not found: %s", visualizationPath)
		}

	
		performancePath := filepath.Join(webRoot, "internal", "interfaces", "web", "templates", "performance_dashboard.html")
		if _, err := os.Stat(performancePath); err == nil {
			r.logger.Infof("Railway: Registering performance dashboard at /performance")
			router.HandleFunc("/performance", func(w http.ResponseWriter, req *http.Request) {
				r.logger.Info("Railway: Serving performance dashboard")
				http.ServeFile(w, req, performancePath)
			}).Methods("GET")
		} else {
			r.logger.Warnf("Railway: Performance dashboard template not found: %s", performancePath)
		}

	
		if _, err := os.Stat(visualizationPath); err == nil {
			r.logger.Info("Railway: Registering root visualization endpoint at /")
			router.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				if req.URL.Path != "/" {
					http.NotFound(w, req)
					return
				}
				r.logger.Info("Railway: Serving main visualization page")
				http.ServeFile(w, req, visualizationPath)
			}).Methods("GET")
		} else {
			r.logger.Warnf("Railway: Root visualization template not found: %s", visualizationPath)
		}
	}

	r.logger.Info("Railway: Successfully registered all visualization routes")
	return nil
}


func (r *RailwayDeployment) findProjectRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		r.logger.Errorf("Railway: Cannot get working directory: %v", err)
		return ""
	}

	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			r.logger.Infof("Railway: Found project root: %s", wd)
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			r.logger.Warn("Railway: Cannot find project root directory")
			return ""
		}
		wd = parent
	}
}


func (r *RailwayDeployment) GetHomepageHandler() http.HandlerFunc {
	if !r.isRailwayEnvironment() {
		return nil
	}
	
	return func(w http.ResponseWriter, req *http.Request) {
		r.logger.Info("Railway: Root endpoint requested - serving Railway homepage")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		html := `<!DOCTYPE html>
<html>
<head>
    <title>SQL Graph Visualizer - Railway</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; 
            max-width: 1000px; 
            margin: 0 auto; 
            padding: 2rem; 
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            color: white;
        }
        .container { 
            background: rgba(255,255,255,0.1); 
            backdrop-filter: blur(10px);
            padding: 2rem; 
            border-radius: 20px; 
            box-shadow: 0 8px 32px rgba(0,0,0,0.3);
            border: 1px solid rgba(255,255,255,0.1);
        }
        h1 { 
            color: white; 
            text-align: center;
            font-size: 2.5rem;
            margin-bottom: 0.5rem;
        }
        .subtitle {
            text-align: center;
            opacity: 0.8;
            margin-bottom: 2rem;
        }
        .status { 
            background: rgba(76, 175, 80, 0.2);
            padding: 1rem; 
            border-left: 4px solid #4CAF50; 
            margin: 1.5rem 0;
            border-radius: 5px;
        }
        .info { 
            background: rgba(33, 150, 243, 0.2);
            padding: 1rem; 
            border-left: 4px solid #2196F3; 
            margin: 1.5rem 0;
            border-radius: 5px;
        }
        .api-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
            gap: 1rem;
            margin: 2rem 0;
        }
        .api-card {
            background: rgba(255,255,255,0.1);
            padding: 1.5rem;
            border-radius: 10px;
            border: 1px solid rgba(255,255,255,0.1);
        }
        .api-card h3 {
            margin: 0 0 0.5rem 0;
            color: #FFD54F;
        }
        a { 
            color: #FFD54F; 
            text-decoration: none;
            font-weight: 500;
        }
        a:hover { 
            text-decoration: underline;
            color: #FFF176;
        }
        code { 
            background: rgba(0,0,0,0.3); 
            padding: 0.2rem 0.5rem; 
            border-radius: 3px;
            font-size: 0.9rem;
        }
        .footer {
            text-align: center;
            margin-top: 3rem;
            opacity: 0.7;
            font-size: 0.9rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>SQL Graph Visualizer</h1>
        <p class="subtitle">Railway Production Deployment</p>
        
        <div class="status">
            <strong>Status:</strong> Application is running successfully on Railway!
        </div>
        
        <div class="info">
            <strong>Platform:</strong> ` + r.GetPlatformName() + `<br>
            <strong>Mode:</strong> Full visualization mode with integrated endpoints<br>
            <strong>Features:</strong> Graph visualization, Performance monitoring, API endpoints
        </div>

        <div class="api-grid">
            <div class="api-card">
                <h3>Health Check</h3>
                <p><code><a href="/api/health">/api/health</a></code></p>
                <p>Monitor application status and database connections</p>
            </div>
            
            <div class="api-card">
                <h3>Debug Info</h3>
                <p><code><a href="/api/debug">/api/debug</a></code></p>
                <p>Technical debugging information and environment details</p>
            </div>
            
            <div class="api-card">
                <h3>Graph Visualization</h3>
                <p><code><a href="/visualization">/visualization</a></code></p>
                <p>Interactive graph visualization interface</p>
            </div>
            
            <div class="api-card">
                <h3>Performance Dashboard</h3>
                <p><code><a href="/performance">/performance</a></code></p>
                <p>Real-time performance monitoring and metrics</p>
            </div>
            
            <div class="api-card">
                <h3>Graph Data API</h3>
                <p><code><a href="/api/graph">/api/graph</a></code></p>
                <p>JSON endpoint for graph nodes and relationships</p>
            </div>
            
            <div class="api-card">
                <h3>⚙️ GraphQL</h3>
                <p><code><a href="/api/graphql">/api/graphql</a></code></p>
                <p>Interactive GraphQL playground for data queries</p>
            </div>
            
            <div class="api-card">
                <h3>Configuration</h3>
                <p><code><a href="/config">/config</a></code></p>
                <p>Application configuration and transform rules</p>
            </div>
        </div>
        
        <div class="info">
            <strong>External Links:</strong><br>
            • <a href="https://github.com/petrms/sql-graph-visualizer" target="_blank">GitHub Repository</a><br>
            • <a href="https://railway.app" target="_blank">Deployed on Railway</a><br>
            • <a href="/api/health">API Status</a>
        </div>
        
        <div class="footer">
            SQL Graph Visualizer v1.1.0 | Railway Deployment<br>
            DDD Architecture with Integrated Visualization
        </div>
    </div>
</body>
</html>`

		if _, err := w.Write([]byte(html)); err != nil {
			r.logger.Errorf("Railway: Error writing homepage response: %v", err)
		}
	}
}
