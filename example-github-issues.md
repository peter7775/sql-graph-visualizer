# Ready-to-use GitHub Issues

Tyto issues můžete zkopírovat a vytvořit přímo na GitHubu:

## Issue #1: Add Loading Indicators During Database Transformation

**Labels:** `good-first-issue`, `frontend`, `enhancement`, `beginner-friendly`

**Description:**
Currently, users don't see any visual feedback during database transformation, which can take several seconds or minutes for large datasets.

**Problem:**
- Users are unsure if the transformation is working
- No progress indication leads to poor user experience
- May result in users refreshing the page or thinking the app is broken

**Proposed Solution:**
Add loading indicators with progress information during:
1. MySQL connection testing
2. Data extraction from MySQL
3. Neo4j data insertion
4. Graph visualization loading

**Acceptance Criteria:**
- [ ] Show spinning loader during transformation
- [ ] Display current step (e.g., "Connecting to MySQL...", "Extracting data...", "Creating nodes...")  
- [ ] Add progress bar for longer operations
- [ ] Disable transformation button during process
- [ ] Show success/error message when complete

**Technical Details:**
- Frontend files to modify: `internal/interfaces/web/`
- May need to add WebSocket for real-time updates
- Use existing logging to track progress steps

**Resources:**
- Current transformation code: `internal/application/services/`
- Web interface: `internal/interfaces/web/`

**Estimated Effort:** 4-6 hours

---

## Issue #2: Add More Configuration Examples

**Labels:** `good-first-issue`, `documentation`, `help-wanted`

**Description:**
The project needs more YAML configuration examples to help users understand different transformation scenarios.

**Current State:**
- Only basic user/team example in README
- Missing examples for common database patterns

**Needed Examples:**
1. **E-commerce database**: Products, categories, orders, customers
2. **Blog system**: Users, posts, comments, tags
3. **CRM system**: Companies, contacts, deals, activities
4. **Social network**: Users, friendships, posts, likes
5. **Inventory management**: Items, suppliers, warehouses

**Acceptance Criteria:**
- [ ] Create `docs/examples/` directory
- [ ] Add 5 complete configuration examples with sample SQL schemas
- [ ] Include README explaining each example
- [ ] Add visualization screenshots for each example
- [ ] Update main README with links to examples

**Files to Create:**
```
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
```

**Technical Requirements:**
- Valid YAML syntax
- Follow existing configuration schema
- Include both node and relationship rules

**Estimated Effort:** 6-8 hours

---

## Issue #3: Implement PostgreSQL Support

**Labels:** `feature`, `backend`, `database`, `medium-difficulty`

**Description:**
Add PostgreSQL as a source database option alongside MySQL.

**Why This Feature?**
- PostgreSQL is widely used in enterprise environments
- Many users have requested this feature
- Expands project's usability significantly

**Technical Requirements:**

1. **Database Driver**
   - Add `github.com/lib/pq` dependency
   - Create PostgreSQL adapter in `internal/infrastructure/persistence/`

2. **Configuration**
   - Extend config schema to support PostgreSQL connection
   - Add database type selection (mysql/postgresql)

3. **SQL Compatibility**
   - Handle PostgreSQL-specific SQL syntax differences
   - Update query builders for PostgreSQL compatibility

4. **Testing**
   - Add PostgreSQL to docker-compose.test.yml
   - Create integration tests with PostgreSQL

**Acceptance Criteria:**
- [ ] PostgreSQL connection established
- [ ] All existing transformation rules work with PostgreSQL
- [ ] Configuration supports both MySQL and PostgreSQL
- [ ] Integration tests pass with PostgreSQL
- [ ] Documentation updated with PostgreSQL examples
- [ ] Docker compose includes PostgreSQL option

**Files to Modify:**
- `internal/infrastructure/persistence/` - Add PostgreSQL repository
- `config/` - Update configuration schema
- `docker-compose.test.yml` - Add PostgreSQL service
- `README.md` - Update documentation

**Resources:**
- [PostgreSQL Go driver documentation](https://pkg.go.dev/github.com/lib/pq)
- Existing MySQL implementation: `internal/infrastructure/persistence/mysql/`

**Estimated Effort:** 2-3 days

---

## Issue #4: Add Dark Mode Theme Toggle

**Labels:** `good-first-issue`, `frontend`, `ui/ux`, `enhancement`

**Description:**
Add a dark/light theme toggle to improve user experience, especially for developers who prefer dark interfaces.

**Current State:**
- Only light theme available
- No theme persistence

**Requirements:**

1. **Theme Toggle**
   - Add toggle button in the header/navbar
   - Switch between light and dark themes
   - Smooth transition animation

2. **Dark Theme Design**
   - Dark background colors
   - Light text colors
   - Maintain good contrast ratios
   - Update graph visualization colors for dark theme

3. **Persistence**
   - Remember user's theme preference in localStorage
   - Apply saved theme on page load

4. **Graph Integration**
   - Update Neovis.js styling for dark theme
   - Ensure node/relationship colors work in both themes

**Acceptance Criteria:**
- [ ] Toggle button added to interface
- [ ] Dark theme fully implemented
- [ ] Theme preference persists across sessions
- [ ] Graph visualization adapts to theme
- [ ] Smooth transition animations
- [ ] All UI elements readable in both themes

**Technical Details:**
- Modify CSS/JS files in `internal/interfaces/web/`
- Use CSS custom properties (variables) for theme colors
- May need to update Neovis.js configuration

**Design Suggestions:**
```css
:root {
  --bg-primary: #ffffff;
  --text-primary: #333333;
  --accent: #2196F3;
}

[data-theme="dark"] {
  --bg-primary: #1a1a1a;
  --text-primary: #ffffff;
  --accent: #64B5F6;
}
```

**Estimated Effort:** 3-4 hours

---

## Issue #5: Add Unit Tests for Domain Layer

**Labels:** `good-first-issue`, `testing`, `code-quality`, `backend`

**Description:**
Increase test coverage in the domain layer to ensure business logic reliability.

**Current State:**
- Limited test coverage in domain layer
- Some domain entities lack proper testing

**Testing Requirements:**

1. **Domain Entities**
   - Test entity creation and validation
   - Test entity methods and business rules
   - Edge cases and error conditions

2. **Domain Models**
   - Test transformation rules parsing
   - Test configuration validation
   - Test data mapping logic

3. **Test Coverage Goals**
   - Achieve 80%+ coverage in domain layer
   - All public methods tested
   - Critical business logic covered

**Files Needing Tests:**
```
internal/domain/
├── entities/
│   ├── transformation_rule.go (needs tests)
│   ├── database_connection.go (needs tests)
│   └── graph_node.go (needs tests)
└── models/
    ├── config.go (needs tests)
    └── mapping.go (needs tests)
```

**Acceptance Criteria:**
- [ ] Create test files for all domain entities
- [ ] Test all public methods and functions  
- [ ] Include edge cases and error scenarios
- [ ] Achieve minimum 80% test coverage
- [ ] All tests pass in CI pipeline
- [ ] Add table-driven tests where appropriate

**Technical Requirements:**
- Use testify framework (already included)
- Follow existing test patterns
- Use mocks for external dependencies
- Write descriptive test names

**Example Test Structure:**
```go
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
```

**Estimated Effort:** 4-6 hours

---

## Issue #6: Improve Error Messages for Better User Experience

**Labels:** `good-first-issue`, `frontend`, `ui/ux`, `beginner-friendly`

**Description:**
Make error messages more user-friendly and actionable.

**Current Problems:**
- Technical error messages confusing to users
- No suggestions for fixing issues
- Stack traces shown to end users

**Examples of Current vs Improved Messages:**

**Current:** `Error: sql: database is locked`
**Improved:** `❌ Database Connection Failed - The database appears to be in use by another process. Please ensure no other applications are accessing the database and try again.`

**Current:** `Error: YAML: line 15: found character that cannot start any token`  
**Improved:** `❌ Configuration Error - There's a syntax error in your configuration file at line 15. Please check for missing quotes or incorrect indentation.`

**Requirements:**

1. **Error Categorization**
   - Database connection errors
   - Configuration file errors  
   - Transformation errors
   - Network/timeout errors

2. **User-Friendly Format**
   - Clear, non-technical language
   - Actionable suggestions
   - Visual indicators (icons, colors)
   - Hide technical details by default (with "Show Details" option)

3. **Error Context**
   - Show which step failed
   - Highlight problematic configuration
   - Provide relevant documentation links

**Acceptance Criteria:**
- [ ] Create error message templates for common scenarios
- [ ] Replace technical errors with user-friendly messages
- [ ] Add actionable suggestions to error messages
- [ ] Include visual indicators (icons, colors)
- [ ] Add "Show Technical Details" toggle
- [ ] Test error handling in different scenarios

**Files to Modify:**
- `internal/application/services/` - Update error handling
- `internal/interfaces/web/` - Update frontend error display
- Create error message templates/constants

**Estimated Effort:** 3-4 hours

---

# Jak vytvořit tyto issues na GitHubu:

1. Jděte na https://github.com/peter7775/mysql-graph-visualizer/issues
2. Klikněte "New Issue"
3. Zkopírujte obsah každého issue
4. Přidejte příslušné labels
5. Assignujte sebe jako maintainer pro mentoring

## Doporučené pořadí vytváření:

1. **Loading Indicators** - Rychlé vylepšení UX
2. **Configuration Examples** - Pomůže novým uživatelům
3. **Error Messages** - Zlepší celkový UX
4. **Dark Mode** - Populární feature
5. **Unit Tests** - Důležité pro kvalitu kódu
6. **PostgreSQL Support** - Větší feature pro pokročilé contributory
