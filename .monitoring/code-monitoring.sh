#!/bin/bash

# SQL Graph Visualizer - Code Monitoring Script
# Monitors for potential unauthorized code copying
# Author: Petr Miroslav Stepanek

set -e

# Configuration
LOG_FILE="monitoring/monitoring-results.log"
DATE=$(date '+%Y-%m-%d %H:%M:%S')
RESULTS_DIR="monitoring/results"

# Create directories if they don't exist
mkdir -p .monitoring/results

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}[${DATE}] Starting Code Monitoring...${NC}"

# Function to log results
log_result() {
    echo "[$DATE] $1" >> "$LOG_FILE"
    echo -e "$1"
}

# Function to search GitHub using curl and GitHub API
search_github() {
    local query="$1"
    local search_type="$2"  # code, repositories
    local output_file="$3"
    
    echo -e "${YELLOW}Searching GitHub for: $query${NC}"
    
    # GitHub API search (requires no auth for basic searches, but has rate limits)
    local api_url="https://api.github.com/search/${search_type}?q=${query}&sort=indexed&order=desc"
    
    curl -s -H "Accept: application/vnd.github.v3+json" \
         "$api_url" > "$output_file" 2>/dev/null || {
        echo -e "${RED}Error accessing GitHub API${NC}"
        return 1
    }
    
    # Parse results and check for matches
    local total_count=$(jq -r '.total_count // 0' "$output_file" 2>/dev/null || echo "0")
    
    if [ "$total_count" -gt 0 ]; then
        echo -e "${RED}⚠️  Found $total_count potential matches for: $query${NC}"
        log_result "ALERT: Found $total_count matches for query: $query"
        
        # Extract repository names and URLs
        if command -v jq >/dev/null 2>&1; then
            jq -r '.items[]? | "\(.html_url // .repository.html_url) - \(.name // .repository.name)"' "$output_file" 2>/dev/null || echo "Could not parse results"
        fi
    else
        echo -e "${GREEN}✓ No matches found for: $query${NC}"
    fi
    
    return 0
}

# Function to search using web scraping (backup method)
search_web() {
    local query="$1"
    local site="$2"  # github.com, google.com
    
    echo -e "${YELLOW}Web searching $site for: $query${NC}"
    
    # Create a simple curl request to check if content exists
    local search_url
    case $site in
        "github")
            search_url="https://github.com/search?q=$(echo "$query" | sed 's/ /+/g')"
            ;;
        "google")
            search_url="https://www.google.com/search?q=$(echo "$query" | sed 's/ /+/g')+site:github.com"
            ;;
    esac
    
    # Simple check if results contain our content
    if curl -s --max-time 10 "$search_url" | grep -qi "sql-graph-visualizer" && \
       ! curl -s --max-time 10 "$search_url" | grep -qi "petrstepanek99"; then
        echo -e "${RED}⚠️  Potential unauthorized copy found via $site${NC}"
        log_result "ALERT: Potential unauthorized copy detected via $site search for: $query"
    fi
}

# Main .monitoring queries
echo -e "${GREEN}=== GitHub API Monitoring ===${NC}"

# High-priority queries (most unique identifiers)
HIGH_PRIORITY_QUERIES=(
    "\"Petr Miroslav Stepanek\" \"petrstepanek99@gmail.com\""
    "\"TransformService struct\" \"databasePort ports.DatabasePort\""
    "\"patent-pending innovations in database analysis\""
    "\"sql-graph-visualizer/internal/application/ports\""
    "\"NewTransformService\" \"Neo4jPort\""
)

# Medium-priority queries
MEDIUM_PRIORITY_QUERIES=(
    "\"GraphAggregate\" \"Neo4jPort\" language:go"
    "\"Domain Driven Design\" \"rule-based transformation\""
    "\"First pass: Creating nodes\" \"Second pass: Creating relationships\""
    "\"convertMapProperties\" \"transform_agg\""
)

# Process high-priority queries
echo -e "${YELLOW}Checking high-priority patterns...${NC}"
for query in "${HIGH_PRIORITY_QUERIES[@]}"; do
    search_github "$(echo "$query" | sed 's/"/%22/g' | sed 's/ /%20/g')" "code" "$RESULTS_DIR/high_$(echo "$query" | md5sum | cut -d' ' -f1).json"
    sleep 2  # Rate limiting
done

# Process medium-priority queries
echo -e "${YELLOW}Checking medium-priority patterns...${NC}"
for query in "${MEDIUM_PRIORITY_QUERIES[@]}"; do
    search_github "$(echo "$query" | sed 's/"/%22/g' | sed 's/ /%20/g')" "code" "$RESULTS_DIR/med_$(echo "$query" | md5sum | cut -d' ' -f1).json"
    sleep 2  # Rate limiting
done

# Repository-level search
echo -e "${YELLOW}Checking for repository copies...${NC}"
search_github "sql-graph-visualizer" "repositories" "$RESULTS_DIR/repo_search.json"

# Generate summary report
echo -e "${GREEN}=== Monitoring Summary ===${NC}"
ALERT_COUNT=$(grep -c "ALERT:" "$LOG_FILE" 2>/dev/null || echo "0")

if [ "$ALERT_COUNT" -gt 0 ]; then
    echo -e "${RED}⚠️  $ALERT_COUNT potential issues detected!${NC}"
    echo -e "${RED}Check $LOG_FILE for details${NC}"
else
    echo -e "${GREEN}✓ No suspicious activity detected${NC}"
fi

# Cleanup old result files (keep last 30 days)
find "$RESULTS_DIR" -name "*.json" -mtime +30 -delete 2>/dev/null || true

log_result "Monitoring completed. Alerts: $ALERT_COUNT"

# Optional: Send email alert if issues found
if [ "$ALERT_COUNT" -gt 0 ] && command -v mail >/dev/null 2>&1; then
    echo "Potential code copying detected. Check monitoring log." | \
    mail -s "SQL Graph Visualizer - Code Monitoring Alert" petrstepanek99@gmail.com 2>/dev/null || true
fi

echo -e "${GREEN}[${DATE}] Code monitoring completed.${NC}"
