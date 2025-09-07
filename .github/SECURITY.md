# Security Policy

## Overview

SQL Graph Visualizer is a database transformation and visualization tool that handles potentially sensitive data connections and transformations. We take security seriously and appreciate the security research community's efforts to responsibly disclose vulnerabilities.

## Supported Versions

We actively maintain security updates for the following versions:

| Version | Supported          | Status |
| ------- | ------------------ | ------ |
| 2.x.x   | :white_check_mark: | Active Development |
| 1.x.x   | :x:                | End of Life |

## Security Considerations

### Data Handling

- **Database Credentials**: This application requires access to source databases (MySQL/PostgreSQL) and target Neo4j instances
- **Data Transformation**: SQL data is processed and transformed into graph format
- **Data Deletion**: The application intentionally deletes all Neo4j data on startup for clean transformations (development/demo behavior)
- **Configuration Files**: Transformation rules and database connections are stored in YAML configuration files

### Network Exposure

- **Web Interface**: Serves visualization interface on port 3000 (configurable)
- **API Endpoints**: Exposes GraphQL (port 8080/graphql) and REST APIs (port 8080/api/*)
- **Database Connections**: Maintains connections to multiple database systems

### Known Security Behaviors

- **Data Cleanup**: Application automatically deletes existing Neo4j data on startup
- **Debug Logging**: May expose sensitive information when LOG_LEVEL=debug is used
- **Configuration Loading**: Supports custom config paths via CONFIG_PATH environment variable

## Reporting Security Vulnerabilities

**Please DO NOT report security vulnerabilities through public GitHub issues.**

### Preferred Reporting Methods

1. **GitHub Security Advisory** (Recommended)
   - Go to <https://github.com/peter7775/sql-graph-visualizer/security/advisories>
   - Click "Report a vulnerability"
   - Provide detailed information about the vulnerability

2. **Direct Email**
   - Send to: security@[your-domain].com
   - Use PGP encryption if possible
   - Include "SQL Graph Visualizer Security" in the subject line

3. **Bug Bounty Platform** (if applicable)
   - Details will be posted here when program is established

### What to Include in Your Report

Please include as much of the following information as possible:

- **Vulnerability Description**: Clear description of the security issue
- **Affected Components**: Which parts of the application are affected
- **Attack Vectors**: How the vulnerability could be exploited
- **Impact Assessment**: Potential impact and severity
- **Proof of Concept**: Steps to reproduce (if safe to include)
- **Suggested Mitigation**: Any ideas for fixing the issue
- **Environment Details**: OS, Go version, database versions, deployment method

### Security Vulnerability Categories

We are particularly interested in reports related to:

#### High Priority

- **SQL Injection**: In transformation rules or query generation
- **Authentication/Authorization Bypass**: Unauthorized access to data or APIs
- **Data Exposure**: Unintended exposure of database credentials or sensitive data
- **Code Injection**: Through configuration files or API inputs
- **Path Traversal**: Via CONFIG_PATH or file operations

#### Medium Priority

- **Cross-Site Scripting (XSS)**: In web visualization interface
- **Cross-Site Request Forgery (CSRF)**: In API endpoints
- **Information Disclosure**: Excessive error messages or debug output
- **Denial of Service**: Resource exhaustion in transformation processes
- **Configuration Issues**: Insecure default settings

#### Lower Priority

- **Dependency Vulnerabilities**: In third-party Go modules
- **Docker Security**: Container configuration issues
- **Documentation**: Security-related documentation improvements

## Response Timeline

We are committed to responding to security reports promptly:

- **Initial Response**: Within 48 hours of report receipt
- **Status Update**: Every 7 days during investigation
- **Resolution Timeline**:
  - Critical vulnerabilities: 7 days
  - High severity: 30 days
  - Medium/Low severity: 90 days

## Security Best Practices for Users

### Deployment Security

- **Never expose the application directly to the internet without proper authentication**
- Use reverse proxy with authentication (nginx, Apache, etc.)
- Configure firewall rules to restrict database access
- Use encrypted connections (TLS) for all database connections
- Regularly update dependencies with `go mod tidy && go mod download`

### Configuration Security

- **Protect configuration files** containing database credentials
- Use environment variables for sensitive configuration values
- Limit filesystem permissions on config directories
- Consider using secret management systems for production deployments

### Database Security

- **Use dedicated database users** with minimal required privileges
- Configure read-only access for source databases when possible
- Regularly review and rotate database credentials
- Monitor database access logs for unusual activity

### Development Security

- **Never commit database credentials** to version control
- Use test databases with non-sensitive data for development
- Set `GO_ENV=test` when running tests to use test configuration
- Review transformation rules for potential injection vulnerabilities

### Docker Security

- Keep base images updated
- Run containers as non-root user when possible
- Use Docker secrets for sensitive configuration
- Regularly scan images for vulnerabilities

## Security Testing

This project includes:

- **Security scanning** with gosec and govulncheck (run via `make sec-scan`)
- **Input validation** in configuration loading
- **Test isolation** with separate test configurations
- **CI/CD security checks** in GitHub Actions

To run security scans locally:

```bash
# Install security tools
make install

# Run comprehensive security scan
make sec-scan

# Run with all CI checks
make ci-check
```

## Acknowledgments

We appreciate security researchers who responsibly disclose vulnerabilities. Contributors will be acknowledged in:

- Security advisory credits
- CHANGELOG.md security section
- Special recognition in project documentation (with permission)

## Contact Information

- **Security Team**: <petrstepanek99@gmail.com>
- **Project Maintainer**: [@peter7775](https://github.com/peter7775)
- **GitHub Security**: <https://github.com/peter7775/sql-graph-visualizer/security>

## Additional Resources

- [Contributing Guidelines](.github/COMMUNITY.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [License Information](LICENSE-AGPL)
- [Project Documentation](README.md)

---

**This security policy was last updated on January 7, 2025.**

For questions about this security policy, please open a [GitHub Discussion](https://github.com/peter7775/sql-graph-visualizer/discussions).
