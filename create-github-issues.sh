#!/bin/bash

# Create GitHub Issues Automatically
# Usage: ./create-github-issues.sh

set -e

REPO="peter7775/mysql-graph-visualizer"

echo "🚀 Creating GitHub issues for contributors..."

# Check if gh CLI is installed and authenticated
if ! command -v gh &> /dev/null; then
    echo "❌ GitHub CLI not found. Please install it first:"
    echo "   https://github.com/cli/cli#installation"
    exit 1
fi

if ! gh auth status &> /dev/null; then
    echo "❌ Not authenticated with GitHub. Please run:"
    echo "   gh auth login"
    exit 1
fi

echo "✅ GitHub CLI is ready!"

# Function to create issue
create_issue() {
    local title="$1"
    local body="$2"
    local labels="$3"
    
    echo "Creating issue: $title"
    
    # Create temporary file for issue body
    local temp_file=$(mktemp)
    echo "$body" > "$temp_file"
    
    gh issue create \
        --repo "$REPO" \
        --title "$title" \
        --body-file "$temp_file" \
        --label "$labels" \
        || echo "⚠️  Could not create issue: $title"
    
    rm "$temp_file"
    echo ""
}

echo "📝 Creating good first issues..."

# Issue 1: Configuration Examples
create_issue "Add More Configuration Examples" "## Description
The project needs more YAML configuration examples to help users understand different transformation scenarios.

## Current State
- Only basic user/team example in README
- Missing examples for common database patterns

## Needed Examples
1. **E-commerce database**: Products, categories, orders, customers
2. **Blog system**: Users, posts, comments, tags  
3. **CRM system**: Companies, contacts, deals, activities
4. **Social network**: Users, friendships, posts, likes
5. **Inventory management**: Items, suppliers, warehouses

## Acceptance Criteria
- [ ] Create \`docs/examples/\` directory
- [ ] Add 5 complete configuration examples with sample SQL schemas
- [ ] Include README explaining each example
- [ ] Add visualization screenshots for each example
- [ ] Update main README with links to examples

## Files to Create
\`\`\`
docs/examples/
├── README.md
├── ecommerce/
│   ├── config.yml
│   ├── sample-schema.sql
│   └── visualization-example.png
├── blog/
├── crm/
├── social-network/
└── inventory/
\`\`\`

## Technical Requirements
- Valid YAML syntax
- Follow existing configuration schema
- Include both node and relationship rules

## Estimated Effort
6-8 hours

## Resources
- Current configuration: \`config/config.yml\`
- Configuration loading: \`internal/config/config.go\`" "good-first-issue,documentation,help-wanted"

# Issue 2: Dark Mode Theme
create_issue "Add Dark Mode Theme Toggle" "## Description
Add a dark/light theme toggle to improve user experience, especially for developers who prefer dark interfaces.

## Current State
- Only light theme available
- No theme persistence

## Requirements

### Theme Toggle
- Add toggle button in the header/navbar
- Switch between light and dark themes
- Smooth transition animation

### Dark Theme Design
- Dark background colors
- Light text colors
- Maintain good contrast ratios
- Update graph visualization colors for dark theme

### Persistence
- Remember user's theme preference in localStorage
- Apply saved theme on page load

### Graph Integration
- Update Neovis.js styling for dark theme
- Ensure node/relationship colors work in both themes

## Acceptance Criteria
- [ ] Toggle button added to interface
- [ ] Dark theme fully implemented
- [ ] Theme preference persists across sessions
- [ ] Graph visualization adapts to theme
- [ ] Smooth transition animations
- [ ] All UI elements readable in both themes

## Technical Details
- Modify CSS/JS files in \`internal/interfaces/web/\`
- Use CSS custom properties (variables) for theme colors
- May need to update Neovis.js configuration

## Design Suggestions
\`\`\`css
:root {
  --bg-primary: #ffffff;
  --text-primary: #333333;
  --accent: #2196F3;
}

[data-theme=\"dark\"] {
  --bg-primary: #1a1a1a;
  --text-primary: #ffffff;
  --accent: #64B5F6;
}
\`\`\`

## Estimated Effort
3-4 hours" "good-first-issue,frontend,enhancement,ui/ux"

# Issue 3: Improve Error Messages
create_issue "Improve Error Messages for Better User Experience" "## Description
Make error messages more user-friendly and actionable.

## Current Problems
- Technical error messages confusing to users
- No suggestions for fixing issues
- Stack traces shown to end users

## Examples of Current vs Improved Messages

**Current:** \`Error: sql: database is locked\`  
**Improved:** \`❌ Database Connection Failed - The database appears to be in use by another process. Please ensure no other applications are accessing the database and try again.\`

**Current:** \`Error: YAML: line 15: found character that cannot start any token\`  
**Improved:** \`❌ Configuration Error - There's a syntax error in your configuration file at line 15. Please check for missing quotes or incorrect indentation.\`

## Requirements

### Error Categorization
- Database connection errors
- Configuration file errors  
- Transformation errors
- Network/timeout errors

### User-Friendly Format
- Clear, non-technical language
- Actionable suggestions
- Visual indicators (icons, colors)
- Hide technical details by default (with \"Show Details\" option)

### Error Context
- Show which step failed
- Highlight problematic configuration
- Provide relevant documentation links

## Acceptance Criteria
- [ ] Create error message templates for common scenarios
- [ ] Replace technical errors with user-friendly messages
- [ ] Add actionable suggestions to error messages
- [ ] Include visual indicators (icons, colors)
- [ ] Add \"Show Technical Details\" toggle
- [ ] Test error handling in different scenarios

## Files to Modify
- \`internal/application/services/\` - Update error handling
- \`internal/interfaces/web/\` - Update frontend error display
- Create error message templates/constants

## Estimated Effort
3-4 hours" "good-first-issue,frontend,ui/ux,beginner-friendly"

# Issue 4: Add Unit Tests for Domain Layer
create_issue "Add Unit Tests for Domain Layer" "## Description
Increase test coverage in the domain layer to ensure business logic reliability.

## Current State
- Limited test coverage in domain layer
- Some domain entities lack proper testing

## Testing Requirements

### Domain Entities
- Test entity creation and validation
- Test entity methods and business rules
- Edge cases and error conditions

### Domain Models
- Test transformation rules parsing
- Test configuration validation
- Test data mapping logic

### Test Coverage Goals
- Achieve 80%+ coverage in domain layer
- All public methods tested
- Critical business logic covered

## Files Needing Tests
\`\`\`
internal/domain/
├── entities/
│   ├── transformation_rule.go (needs tests)
│   ├── database_connection.go (needs tests)
│   └── graph_node.go (needs tests)
└── models/
    ├── config.go (needs tests)
    └── mapping.go (needs tests)
\`\`\`

## Acceptance Criteria
- [ ] Create test files for all domain entities
- [ ] Test all public methods and functions  
- [ ] Include edge cases and error scenarios
- [ ] Achieve minimum 80% test coverage
- [ ] All tests pass in CI pipeline
- [ ] Add table-driven tests where appropriate

## Technical Requirements
- Use testify framework (already included)
- Follow existing test patterns
- Use mocks for external dependencies
- Write descriptive test names

## Example Test Structure
\`\`\`go
func TestTransformationRule_Validate(t *testing.T) {
    tests := []struct {
        name    string
        rule    *TransformationRule
        wantErr bool
    }{
        // test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
\`\`\`

## Estimated Effort
4-6 hours" "good-first-issue,testing,backend,code-quality"

# Issue 5: PostgreSQL Support (Medium difficulty)
create_issue "Implement PostgreSQL Support" "## Description
Add PostgreSQL as a source database option alongside MySQL.

## Why This Feature?
- PostgreSQL is widely used in enterprise environments
- Many users have requested this feature
- Expands project's usability significantly

## Technical Requirements

### Database Driver
- Add \`github.com/lib/pq\` dependency
- Create PostgreSQL adapter in \`internal/infrastructure/persistence/\`

### Configuration
- Extend config schema to support PostgreSQL connection
- Add database type selection (mysql/postgresql)

### SQL Compatibility
- Handle PostgreSQL-specific SQL syntax differences
- Update query builders for PostgreSQL compatibility

### Testing
- Add PostgreSQL to docker-compose.test.yml
- Create integration tests with PostgreSQL

## Acceptance Criteria
- [ ] PostgreSQL connection established
- [ ] All existing transformation rules work with PostgreSQL
- [ ] Configuration supports both MySQL and PostgreSQL
- [ ] Integration tests pass with PostgreSQL
- [ ] Documentation updated with PostgreSQL examples
- [ ] Docker compose includes PostgreSQL option

## Files to Modify
- \`internal/infrastructure/persistence/\` - Add PostgreSQL repository
- \`config/\` - Update configuration schema
- \`docker-compose.test.yml\` - Add PostgreSQL service
- \`README.md\` - Update documentation

## Resources
- [PostgreSQL Go driver documentation](https://pkg.go.dev/github.com/lib/pq)
- Existing MySQL implementation: \`internal/infrastructure/persistence/mysql/\`

## Estimated Effort
2-3 days" "feature,backend,database,medium-difficulty"

# Issue 6: Export Functionality
create_issue "Add Graph Data Export Functionality" "## Description
Allow users to export graph data in various formats for external use and analysis.

## Requirements

### Supported Export Formats
- **JSON** - Complete graph data with nodes and relationships
- **CSV** - Separate files for nodes and relationships
- **GraphML** - Standard graph format for other visualization tools
- **Cypher Script** - Neo4j queries to recreate the graph

### Export Options
- **Full Graph** - Export entire transformed graph
- **Filtered Export** - Export based on node types or properties
- **Query Results** - Export results of specific GraphQL queries

### User Interface
- Add export button to visualization interface
- Format selection dropdown
- Progress indicator for large exports
- Download link generation

## Acceptance Criteria
- [ ] Export button added to UI
- [ ] Support for JSON, CSV, GraphML, Cypher formats
- [ ] Filter options for selective export
- [ ] Progress indication for large datasets
- [ ] Proper error handling for export failures
- [ ] API endpoints for programmatic export

## Technical Implementation

### API Endpoints
\`\`\`
GET /api/export/json
GET /api/export/csv
GET /api/export/graphml
GET /api/export/cypher
\`\`\`

### Query Parameters
- \`format\` - Export format
- \`nodeTypes\` - Filter by node types
- \`limit\` - Limit number of results

## Files to Modify
- \`internal/application/services/\` - Add export service
- \`internal/interfaces/web/\` - Add export endpoints
- Frontend - Add export UI components

## Estimated Effort
1-2 days" "feature,frontend,backend,enhancement"

echo "✨ All issues created successfully!"
echo ""
echo "📝 Next steps:"
echo "1. Check the issues at: https://github.com/$REPO/issues"
echo "2. Add appropriate labels to each issue"
echo "3. Consider creating more advanced issues for experienced contributors"
echo ""
echo "💡 Tips for managing issues:"
echo "• Respond to contributor questions within 24 hours"
echo "• Provide additional guidance when requested"
echo "• Thank contributors for their interest"
echo "• Update issue descriptions based on feedback"
