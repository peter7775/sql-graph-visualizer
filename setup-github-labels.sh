#!/bin/bash

# Setup GitHub Labels for Contributors
# Run with: ./setup-github-labels.sh

set -e

REPO="peter7775/mysql-graph-visualizer"

echo "🚀 Setting up GitHub labels for contributors..."

# Check if gh CLI is installed
if ! command -v gh &> /dev/null; then
    echo "❌ GitHub CLI not found. Please install it first:"
    echo "   https://github.com/cli/cli#installation"
    exit 1
fi

# Check if authenticated
if ! gh auth status &> /dev/null; then
    echo "❌ Not authenticated with GitHub. Please run:"
    echo "   gh auth login"
    exit 1
fi

echo "✅ GitHub CLI is ready!"

# Function to create or update label
create_label() {
    local name="$1"
    local color="$2"
    local description="$3"
    
    echo "Creating label: $name"
    gh label create "$name" --color "$color" --description "$description" --repo "$REPO" 2>/dev/null || \
    gh label edit "$name" --color "$color" --description "$description" --repo "$REPO" 2>/dev/null || \
    echo "⚠️  Could not create/update label: $name"
}

echo ""
echo "🏷️  Creating contributor-friendly labels..."

# Contributor Experience Labels
create_label "good-first-issue" "7057ff" "Good for newcomers - easy to understand and implement"
create_label "help-wanted" "008672" "Extra attention is needed - maintainers are looking for help"
create_label "beginner-friendly" "c2e0c6" "Suitable for developers new to the project or technology"
create_label "mentor-available" "0052cc" "A maintainer is available to provide guidance and support"

echo ""
echo "🎯 Creating skill-based labels..."

# Skill-based Labels  
create_label "documentation" "0075ca" "Improvements or additions to documentation"
create_label "frontend" "d93f0b" "Frontend/UI related work"
create_label "backend" "1d76db" "Backend/server related work" 
create_label "devops" "5319e7" "DevOps, infrastructure, and deployment related"
create_label "testing" "fbca04" "Testing related work - unit tests, integration tests, etc."
create_label "database" "006b75" "Database related changes - MySQL, Neo4j, queries"

echo ""
echo "📊 Creating priority labels..."

# Priority Labels
create_label "priority-high" "d73a4a" "High priority - should be addressed soon"
create_label "priority-medium" "fbca04" "Medium priority - normal timeline"
create_label "priority-low" "0e8a16" "Low priority - nice to have"

echo ""
echo "🔧 Creating type labels..."

# Type Labels
create_label "feature" "a2eeef" "New feature or functionality"
create_label "enhancement" "84b6eb" "Improvement to existing functionality"
create_label "bug" "d73a4a" "Something isn't working correctly"
create_label "refactor" "5319e7" "Code refactoring - no functional changes"
create_label "performance" "1d76db" "Performance improvements"

echo ""
echo "📚 Creating topic labels..."

# Topic-specific Labels
create_label "api" "c5def5" "REST API or GraphQL related"
create_label "visualization" "f9d0c4" "Graph visualization and UI"
create_label "configuration" "d4c5f9" "Configuration files and management"
create_label "docker" "0052cc" "Docker and containerization"
create_label "security" "b60205" "Security related improvements"

echo ""
echo "🎉 Creating experience level labels..."

# Experience Level Labels
create_label "level-beginner" "c2e0c6" "Requires basic programming knowledge"
create_label "level-intermediate" "fbca04" "Requires some experience with the technology stack"
create_label "level-advanced" "d73a4a" "Requires deep understanding of the system"

echo ""
echo "✨ All labels created successfully!"
echo ""
echo "📝 Next steps:"
echo "1. Go to https://github.com/$REPO/issues"
echo "2. Create new issues using the examples from example-github-issues.md"
echo "3. Apply appropriate labels to each issue"
echo ""
echo "💡 Pro tips for attracting contributors:"
echo "• Start with 3-5 good-first-issue items"
echo "• Respond quickly to contributor questions"
echo "• Be welcoming and encouraging in discussions"
echo "• Provide clear code review feedback"
echo "• Thank contributors publicly"
echo ""
echo "🌟 Your project is now ready to attract contributors!"
