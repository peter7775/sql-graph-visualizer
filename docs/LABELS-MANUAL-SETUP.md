#  Manual Label Setup Guide

If you prefer to set up labels manually or the automated script doesn't work, you can create them directly in GitHub's web interface.

##  How to Create Labels Manually

1. Go to your repository on GitHub
2. Click on **"Issues"** tab
3. Click on **"Labels"** (next to Milestones)
4. Click **"New label"** button
5. Fill in the details from the tables below

---

## ️ Essential Labels to Create First

Start with these core labels that are used by your issue templates:

### **Priority Labels**
```
priority:critical - #b60205 - Critical priority - needs immediate attention
priority:high     - #d93f0b - High priority  
priority:medium   - #fbca04 - Medium priority
priority:low      - #0e8a16 - Low priority
```

### **Status Labels**
```
needs-triage   - #ededed - Issue needs to be triaged and prioritized
needs-review   - #fbca04 - Waiting for code review
ready-to-merge - #0e8a16 - Approved and ready for merge
blocked        - #b60205 - Progress is blocked by external dependency
```

### **Technology Labels**
```
mysql         - #e97627 - MySQL database related
neo4j         - #4581ea - Neo4j graph database related
golang        - #00add8 - Go programming language
docker        - #0db7ed - Docker containerization
visualization - #8a2be2 - Graph visualization and UI
```

### **Component Labels (DDD Architecture)**
```
domain         - #5319e7 - Domain layer - core business logic
application    - #1d76db - Application layer - use cases and services
infrastructure - #0052cc - Infrastructure layer - databases, external services
interface      - #006b75 - Interface layer - APIs and web interfaces
```

---

##  Complete Label List

Copy and paste these into GitHub's label creation form:

### Type Labels
| Name | Color | Description |
|------|-------|-------------|
| `bug` | `#d73a4a` | Something isn't working |
| `enhancement` | `#a2eeef` | New feature or request |
| `documentation` | `#0075ca` | Improvements or additions to documentation |
| `question` | `#d876e3` | Further information is requested |

### Feature Areas
| Name | Color | Description |
|------|-------|-------------|
| `database` | `#d4c5f9` | Database operations and connections |
| `transformation` | `#c5def5` | Data transformation logic |
| `api` | `#f9c2ff` | REST or GraphQL API endpoints |
| `config` | `#ffdda0` | Configuration management |
| `performance` | `#ff6b6b` | Performance optimization and issues |
| `security` | `#ff9f43` | Security related issues |
| `connection` | `#f1c0e8` | Database connection issues |

### Effort Estimation
| Name | Color | Description |
|------|-------|-------------|
| `effort:small` | `#c2e0c6` | Small effort - 1-2 hours |
| `effort:medium` | `#ffdda0` | Medium effort - 1-2 days |
| `effort:large` | `#ffaaa5` | Large effort - 1+ weeks |
| `effort:epic` | `#d1ecf1` | Epic - multiple weeks/months |

### Special Labels
| Name | Color | Description |
|------|-------|-------------|
| `breaking-change` | `#b60205` | Changes that break backward compatibility |
| `good-first-issue` | `#7057ff` | Good for newcomers |
| `help-wanted` | `#008672` | Extra attention is needed |
| `architecture` | `#5319e7` | Architectural decisions and changes |

---

##  Alternative Setup Methods

### Method 1: GitHub CLI (Recommended)
```bash
# First authenticate
gh auth login

# Then run the script
./scripts/setup-labels.sh
```

### Method 2: GitHub API with curl
```bash
# Set your GitHub token
export GITHUB_TOKEN="your_token_here"
export REPO="username/mysql-graph-visualizer"

# Create a label (example)
curl -X POST \
  -H "Authorization: token $GITHUB_TOKEN" \
  -H "Accept: application/vnd.github.v3+json" \
  https://api.github.com/repos/$REPO/labels \
  -d '{"name":"bug","color":"d73a4a","description":"Something isn'\''t working"}'
```

### Method 3: Import from another repository
If you have labels set up in another repository:
1. Go to the source repository's labels page
2. Use browser extensions like "GitHub Label Manager" 
3. Export and import to your new repository

---

##  Verification Checklist

After creating labels, verify you have:

- [ ] All priority labels (`priority:critical`, `priority:high`, etc.)
- [ ] Status labels (`needs-triage`, `needs-review`, etc.) 
- [ ] Technology labels (`mysql`, `neo4j`, `golang`, etc.)
- [ ] Component labels (`domain`, `application`, `infrastructure`, `interface`)
- [ ] Basic type labels (`bug`, `enhancement`, `documentation`)
- [ ] Effort labels (`effort:small`, `effort:medium`, etc.)

##  Quick Test

Create a test issue and try applying these label combinations:
- `bug + mysql + connection + priority:high`
- `enhancement + visualization + effort:medium`
- `question + documentation + needs-triage`

---

## 🆘 Troubleshooting

**Can't see Labels tab?**
- Make sure you have admin access to the repository
- Labels tab is next to Milestones in the Issues section

**Colors not showing correctly?**
- Remove the `#` when entering colors in GitHub's interface
- Use 6-digit hex codes (e.g., `d73a4a` not `d73a4a`)

**Too many labels to create manually?**
- Start with just the essential labels listed at the top
- Add more categories as needed
- Use the automated script for bulk creation

**Need to delete existing labels?**
- Go to Labels page, click the label name, then "Delete label"
- Be careful not to delete labels that are already in use on issues
