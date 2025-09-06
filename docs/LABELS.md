# ️ GitHub Labels Guide

This document describes the labeling system for the mysql-graph-visualizer project.

##  Quick Setup

Run the setup script to create all labels automatically:

```bash
./scripts/setup-labels.sh
```

**Prerequisites:**
- [GitHub CLI](https://cli.github.com/) installed
- Authenticated with GitHub (`gh auth login`)
- Repository access permissions

---

## ️ Label Categories

###  **Type Labels** (What kind of issue/PR)
| Label | Description | Color |
|-------|-------------|-------|
| `bug` | Something isn't working | ![#d73a4a](https://via.placeholder.com/15/d73a4a/000000?text=+) `#d73a4a` |
| `enhancement` | New feature or request | ![#a2eeef](https://via.placeholder.com/15/a2eeef/000000?text=+) `#a2eeef` |
| `documentation` | Improvements or additions to documentation | ![#0075ca](https://via.placeholder.com/15/0075ca/000000?text=+) `#0075ca` |
| `question` | Further information is requested | ![#d876e3](https://via.placeholder.com/15/d876e3/000000?text=+) `#d876e3` |

###  **Priority Labels** (How urgent)
| Label | Description | Color |
|-------|-------------|-------|
| `priority:critical` | Critical priority - needs immediate attention | ![#b60205](https://via.placeholder.com/15/b60205/000000?text=+) `#b60205` |
| `priority:high` | High priority | ![#d93f0b](https://via.placeholder.com/15/d93f0b/000000?text=+) `#d93f0b` |
| `priority:medium` | Medium priority | ![#fbca04](https://via.placeholder.com/15/fbca04/000000?text=+) `#fbca04` |
| `priority:low` | Low priority | ![#0e8a16](https://via.placeholder.com/15/0e8a16/000000?text=+) `#0e8a16` |

###  **Status Labels** (Current state)
| Label | Description | Color |
|-------|-------------|-------|
| `needs-triage` | Issue needs to be triaged and prioritized | ![#ededed](https://via.placeholder.com/15/ededed/000000?text=+) `#ededed` |
| `needs-review` | Waiting for code review | ![#fbca04](https://via.placeholder.com/15/fbca04/000000?text=+) `#fbca04` |
| `needs-testing` | Requires testing before merge | ![#f9d0c4](https://via.placeholder.com/15/f9d0c4/000000?text=+) `#f9d0c4` |
| `ready-to-merge` | Approved and ready for merge | ![#0e8a16](https://via.placeholder.com/15/0e8a16/000000?text=+) `#0e8a16` |
| `blocked` | Progress is blocked by external dependency | ![#b60205](https://via.placeholder.com/15/b60205/000000?text=+) `#b60205` |
| `wip` | Work in progress | ![#fef2c0](https://via.placeholder.com/15/fef2c0/000000?text=+) `#fef2c0` |

### ️ **Component Labels** (DDD Architecture)
| Label | Description | Color |
|-------|-------------|-------|
| `domain` | Domain layer - core business logic | ![#5319e7](https://via.placeholder.com/15/5319e7/000000?text=+) `#5319e7` |
| `application` | Application layer - use cases and services | ![#1d76db](https://via.placeholder.com/15/1d76db/000000?text=+) `#1d76db` |
| `infrastructure` | Infrastructure layer - databases, external services | ![#0052cc](https://via.placeholder.com/15/0052cc/000000?text=+) `#0052cc` |
| `interface` | Interface layer - APIs and web interfaces | ![#006b75](https://via.placeholder.com/15/006b75/000000?text=+) `#006b75` |

###  **Technology Labels** (Tech stack)
| Label | Description | Color |
|-------|-------------|-------|
| `mysql` | MySQL database related | ![#e97627](https://via.placeholder.com/15/e97627/000000?text=+) `#e97627` |
| `neo4j` | Neo4j graph database related | ![#4581ea](https://via.placeholder.com/15/4581ea/000000?text=+) `#4581ea` |
| `graphql` | GraphQL API related | ![#e10098](https://via.placeholder.com/15/e10098/000000?text=+) `#e10098` |
| `golang` | Go programming language | ![#00add8](https://via.placeholder.com/15/00add8/000000?text=+) `#00add8` |
| `docker` | Docker containerization | ![#0db7ed](https://via.placeholder.com/15/0db7ed/000000?text=+) `#0db7ed` |
| `visualization` | Graph visualization and UI | ![#8a2be2](https://via.placeholder.com/15/8a2be2/000000?text=+) `#8a2be2` |

###  **Feature Area Labels** (Functional areas)
| Label | Description | Color |
|-------|-------------|-------|
| `database` | Database operations and connections | ![#d4c5f9](https://via.placeholder.com/15/d4c5f9/000000?text=+) `#d4c5f9` |
| `transformation` | Data transformation logic | ![#c5def5](https://via.placeholder.com/15/c5def5/000000?text=+) `#c5def5` |
| `api` | REST or GraphQL API endpoints | ![#f9c2ff](https://via.placeholder.com/15/f9c2ff/000000?text=+) `#f9c2ff` |
| `config` | Configuration management | ![#ffdda0](https://via.placeholder.com/15/ffdda0/000000?text=+) `#ffdda0` |
| `performance` | Performance optimization and issues | ![#ff6b6b](https://via.placeholder.com/15/ff6b6b/000000?text=+) `#ff6b6b` |
| `security` | Security related issues | ![#ff9f43](https://via.placeholder.com/15/ff9f43/000000?text=+) `#ff9f43` |

### ⏱️ **Effort Labels** (Time estimation)
| Label | Description | Color |
|-------|-------------|-------|
| `effort:small` | Small effort - 1-2 hours | ![#c2e0c6](https://via.placeholder.com/15/c2e0c6/000000?text=+) `#c2e0c6` |
| `effort:medium` | Medium effort - 1-2 days | ![#ffdda0](https://via.placeholder.com/15/ffdda0/000000?text=+) `#ffdda0` |
| `effort:large` | Large effort - 1+ weeks | ![#ffaaa5](https://via.placeholder.com/15/ffaaa5/000000?text=+) `#ffaaa5` |
| `effort:epic` | Epic - multiple weeks/months | ![#d1ecf1](https://via.placeholder.com/15/d1ecf1/000000?text=+) `#d1ecf1` |

### ⭐ **Special Labels**
| Label | Description | Color |
|-------|-------------|-------|
| `breaking-change` | Changes that break backward compatibility | ![#b60205](https://via.placeholder.com/15/b60205/000000?text=+) `#b60205` |
| `good-first-issue` | Good for newcomers | ![#7057ff](https://via.placeholder.com/15/7057ff/000000?text=+) `#7057ff` |
| `help-wanted` | Extra attention is needed | ![#008672](https://via.placeholder.com/15/008672/000000?text=+) `#008672` |
| `architecture` | Architectural decisions and changes | ![#5319e7](https://via.placeholder.com/15/5319e7/000000?text=+) `#5319e7` |

---

##  **Usage Examples**

### **Issue Labeling Examples:**

**Database Connection Bug:**
```
bug + mysql + connection + priority:high + infrastructure
```

**New Visualization Feature:**
```
enhancement + visualization + neo4j + effort:medium + interface
```

**Performance Issue:**
```
performance + transformation + domain + priority:medium + needs-testing
```

**Configuration Enhancement:**
```
enhancement + config + effort:small + good-first-issue
```

### **Pull Request Labeling Examples:**

**Bug Fix PR:**
```
bug + mysql + infrastructure + ready-to-merge
```

**Breaking Change PR:**
```
enhancement + breaking-change + domain + needs-review
```

**Documentation Update:**
```
documentation + effort:small + ready-to-merge
```

---

##  **Best Practices**

### **For Issues:**
1. **Always use a type label** (`bug`, `enhancement`, `question`, etc.)
2. **Add priority** for bugs and critical features
3. **Include technology** labels for tech-specific issues
4. **Use component labels** to identify architectural layer
5. **Add effort estimation** to help with planning

### **For Pull Requests:**
1. **Match the related issue labels** when possible
2. **Use status labels** to track review progress
3. **Add `breaking-change`** for backward compatibility issues
4. **Include technology and component** labels for context

### **Label Combinations:**
- **Minimum**: Type + Priority/Effort
- **Recommended**: Type + Technology + Component + Priority/Effort
- **Complete**: Type + Technology + Component + Feature Area + Priority/Effort + Status

---

##  **Maintenance**

### **Regular Label Cleanup:**
- Review `stale` labeled issues monthly
- Update priority labels based on project needs
- Archive completed `epic` labels
- Remove outdated technology labels

### **Adding New Labels:**
1. Follow the naming convention: `category:name` or `name`
2. Choose colors that match the category
3. Update this documentation
4. Update the setup script

### **Label Color Scheme:**
- **Red** (`#b60205`, `#d73a4a`): Critical, bugs, blocking
- **Orange** (`#d93f0b`, `#e97627`): High priority, warnings
- **Yellow** (`#fbca04`, `#ffdda0`): Medium priority, in progress
- **Green** (`#0e8a16`, `#c2e0c6`): Low priority, ready, completed
- **Blue** (`#0075ca`, `#1d76db`): Features, documentation
- **Purple** (`#5319e7`, `#8a2be2`): Architecture, special categories

---

##  **Contributing**

When contributing to the project:
1. **Use appropriate labels** on your issues and PRs
2. **Follow the label combinations** suggested above
3. **Update label documentation** if adding new label categories
4. **Be consistent** with existing labeling patterns

For questions about labeling, create an issue with the `question` + `documentation` labels.
