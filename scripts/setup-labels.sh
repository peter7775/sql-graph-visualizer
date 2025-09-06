#!/bin/bash

# GitHub Labels Setup Script for mysql-graph-visualizer
# This script creates all the labels needed for the project using GitHub CLI

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Setting up GitHub labels for mysql-graph-visualizer${NC}"
echo ""

# Check if gh CLI is installed
if ! command -v gh &> /dev/null; then
    echo -e "${RED}ERROR: GitHub CLI (gh) is not installed. Please install it first.${NC}"
    echo "Visit: https://cli.github.com/"
    exit 1
fi

# Check if user is authenticated
if ! gh auth status &> /dev/null; then
    echo -e "${YELLOW}WARNING: Please authenticate with GitHub CLI first:${NC}"
    echo "gh auth login"
    exit 1
fi

echo -e "${YELLOW}Creating labels...${NC}"
echo ""

# Function to create label
create_label() {
    local name="$1"
    local description="$2"
    local color="$3"
    
    if gh label create "$name" --description "$description" --color "$color" 2>/dev/null; then
        echo -e "${GREEN}Created: $name${NC}"
    else
        echo -e "${YELLOW}WARNING: Label '$name' already exists, skipping...${NC}"
    fi
}

# TYPE LABELS
echo -e "${BLUE}Type Labels${NC}"
create_label "bug" "Something isn't working" "d73a4a"
create_label "enhancement" "New feature or request" "a2eeef"
create_label "documentation" "Improvements or additions to documentation" "0075ca"
create_label "question" "Further information is requested" "d876e3"
create_label "duplicate" "This issue or pull request already exists" "cfd3d7"
create_label "invalid" "This doesn't seem right" "e4e669"
create_label "wontfix" "This will not be worked on" "ffffff"

# PRIORITY LABELS
echo -e "${BLUE}Priority Labels${NC}"
create_label "priority:critical" "Critical priority - needs immediate attention" "b60205"
create_label "priority:high" "High priority" "d93f0b"
create_label "priority:medium" "Medium priority" "fbca04"
create_label "priority:low" "Low priority" "0e8a16"

# STATUS LABELS
echo -e "${BLUE}Status Labels${NC}"
create_label "needs-triage" "Issue needs to be triaged and prioritized" "ededed"
create_label "needs-review" "Waiting for code review" "fbca04"
create_label "needs-testing" "Requires testing before merge" "f9d0c4"
create_label "ready-to-merge" "Approved and ready for merge" "0e8a16"
create_label "blocked" "Progress is blocked by external dependency" "b60205"
create_label "wip" "Work in progress" "fef2c0"
create_label "stale" "No recent activity" "ededed"

# COMPONENT LABELS (Architecture specific)
echo -e "${BLUE}Component Labels${NC}"
create_label "domain" "Domain layer - core business logic" "5319e7"
create_label "application" "Application layer - use cases and services" "1d76db"
create_label "infrastructure" "Infrastructure layer - databases, external services" "0052cc"
create_label "interface" "Interface layer - APIs and web interfaces" "006b75"

# TECHNOLOGY LABELS
echo -e "${BLUE}Technology Labels${NC}"
create_label "mysql" "MySQL database related" "e97627"
create_label "neo4j" "Neo4j graph database related" "4581ea"
create_label "graphql" "GraphQL API related" "e10098"
create_label "golang" "Go programming language" "00add8"
create_label "docker" "Docker containerization" "0db7ed"
create_label "visualization" "Graph visualization and UI" "8a2be2"

# FEATURE AREA LABELS
echo -e "${BLUE}Feature Area Labels${NC}"
create_label "database" "Database operations and connections" "d4c5f9"
create_label "transformation" "Data transformation logic" "c5def5"
create_label "api" "REST or GraphQL API endpoints" "f9c2ff"
create_label "config" "Configuration management" "ffdda0"
create_label "logging" "Logging and monitoring" "d1ecf1"
create_label "testing" "Testing infrastructure and test cases" "c2e0c6"
create_label "performance" "Performance optimization and issues" "ff6b6b"
create_label "security" "Security related issues" "ff9f43"

# CONNECTION/ISSUE TYPE LABELS
echo -e "${BLUE}Connection & Issue Type Labels${NC}"
create_label "connection" "Database connection issues" "f1c0e8"
create_label "migration" "Data migration related" "c0f0ea"
create_label "visualization-bug" "Graph visualization rendering issues" "ffaaa5"
create_label "query-performance" "Database query performance issues" "ff8b94"

# EFFORT LABELS
echo -e "${BLUE}Effort Labels${NC}"
create_label "effort:small" "Small effort - 1-2 hours" "c2e0c6"
create_label "effort:medium" "Medium effort - 1-2 days" "ffdda0"
create_label "effort:large" "Large effort - 1+ weeks" "ffaaa5"
create_label "effort:epic" "Epic - multiple weeks/months" "d1ecf1"

# SPECIAL LABELS
echo -e "${BLUE}Special Labels${NC}"
create_label "breaking-change" "Changes that break backward compatibility" "b60205"
create_label "good-first-issue" "Good for newcomers" "7057ff"
create_label "help-wanted" "Extra attention is needed" "008672"
create_label "architecture" "Architectural decisions and changes" "5319e7"

echo ""
echo -e "${GREEN}Labels setup completed!${NC}"
echo ""
echo -e "${YELLOW}Usage Tips:${NC}"
echo "- Use 'needs-triage' for new issues that need prioritization"
echo "- Combine type + component + technology labels for better organization"
echo "- Use priority labels to manage your backlog"
echo "- Apply 'breaking-change' label to PRs that affect compatibility"
echo ""
echo -e "${BLUE}Example label combinations:${NC}"
echo "• bug + mysql + connection + priority:high"
echo "• enhancement + visualization + neo4j + effort:medium"
echo "• performance + transformation + domain + needs-testing"
echo ""
