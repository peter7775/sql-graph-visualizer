#!/bin/bash

# GitHub Issues Management Helper
# Usage: ./manage-issues.sh [command]

set -e

REPO="peter7775/sql-graph-visualizer"

case "${1:-help}" in
    "list")
        echo "📋 Current GitHub Issues:"
        echo ""
        gh issue list --repo "$REPO"
        ;;
        
    "good-first")
        echo "🟢 Good First Issues:"
        echo ""
        gh issue list --repo "$REPO" --label "good-first-issue"
        ;;
        
    "stats")
        echo "📊 Issue Statistics:"
        echo ""
        total=$(gh issue list --repo "$REPO" --json number --jq '. | length')
        good_first=$(gh issue list --repo "$REPO" --label "good-first-issue" --json number --jq '. | length')
        help_wanted=$(gh issue list --repo "$REPO" --label "help-wanted" --json number --jq '. | length')
        beginner=$(gh issue list --repo "$REPO" --label "beginner-friendly" --json number --jq '. | length')
        
        echo "Total Issues: $total"
        echo "Good First Issues: $good_first"
        echo "Help Wanted: $help_wanted"
        echo "Beginner Friendly: $beginner"
        ;;
        
    "labels")
        echo "🏷️  Available Labels:"
        echo ""
        gh label list --repo "$REPO"
        ;;
        
    "add-mentor")
        if [ -z "$2" ]; then
            echo "Usage: $0 add-mentor <issue_number>"
            exit 1
        fi
        echo "Adding mentor-available label to issue #$2"
        gh issue edit "$2" --add-label "mentor-available" --repo "$REPO"
        ;;
        
    "promote")
        echo "🚀 Promoting Good First Issues:"
        echo ""
        echo "Links to share:"
        echo "• Good First Issues: https://github.com/$REPO/labels/good-first-issue"
        echo "• Help Wanted: https://github.com/$REPO/labels/help-wanted"
        echo "• All Issues: https://github.com/$REPO/issues"
        echo ""
        echo "External platforms:"
        echo "• up-for-grabs.net: Add project to their list"
        echo "• goodfirstissue.dev: Submit your repository"
        echo "• Reddit post: Share on r/golang, r/opensource"
        ;;
        
    "template")
        echo "📝 Creating new issue template:"
        echo ""
        cat << 'EOF'
## Description
[Clear description of what needs to be done]

## Current State
[What's the current situation?]

## Requirements
[What are the specific requirements?]

## Acceptance Criteria
- [ ] Criterion 1
- [ ] Criterion 2
- [ ] Criterion 3

## Technical Details
[Any technical guidance or constraints]

## Files to Modify
- `file1.go` - Description
- `file2.js` - Description

## Resources
- [Link to relevant documentation]
- [Link to similar implementation]

## Estimated Effort
[Time estimate: hours/days]

## Prerequisites (if any)
[Knowledge or experience required]
EOF
        ;;
        
    "help"|*)
        echo "🔧 GitHub Issues Management Helper"
        echo ""
        echo "Commands:"
        echo "  list         - Show all open issues"
        echo "  good-first   - Show only good-first-issue issues"
        echo "  stats        - Show issue statistics"
        echo "  labels       - List all labels"
        echo "  add-mentor   - Add mentor-available label to specific issue"
        echo "  promote      - Show promotion links and tips"
        echo "  template     - Show issue template"
        echo "  help         - Show this help"
        echo ""
        echo "Examples:"
        echo "  ./manage-issues.sh list"
        echo "  ./manage-issues.sh good-first"
        echo "  ./manage-issues.sh add-mentor 5"
        ;;
esac
