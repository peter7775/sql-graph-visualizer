# Troubleshooting Guide

This document contains solutions to common issues you might encounter with the mysql-graph-visualizer project.

## GitHub Actions Issues

### Deployment Status Update Error

**Problem**: GitHub Actions workflow fails with error:
```
RequestError [HttpError]: Not Found
Error: Unhandled error: HttpError: Not Found
...
deployment_id: context.payload.deployment?.id || 0
```

**Cause**: The `update-status` job in the deploy workflow was trying to update a deployment status using a deployment ID that doesn't exist (defaulting to `0`).

**Solution**: The issue has been fixed by replacing the problematic deployment status update with a simple logging mechanism that creates a GitHub Actions summary instead.

**Fixed Files**:
- `.github/workflows/deploy.yml` - Updated `update-status` job to use logging instead of GitHub API calls

### gosec Installation Issues in CI

**Problem**: CI pipeline fails with:
```
go: github.com/securecodewarrior/gosec/v2/cmd/gosec@latest: git ls-remote -q origin
fatal: could not read Username for 'https://github.com': terminal prompts disabled
```

**Cause**: GitHub Actions cannot authenticate to install Go modules via `go install`.

**Solution**: Use binary installation instead of `go install`. This has been fixed in the workflow.

**Fixed Files**:
- `.github/workflows/go.yml` - Updated to use curl-based binary installation

### Workflow Validation

To validate your GitHub Actions workflows locally, use:
```bash
./scripts/validate-workflows.sh
```

This script checks all workflow files for YAML syntax errors.

## Security Issues

### Running Security Scans

Use the provided security check script:
```bash
./scripts/security-check.sh
```

This runs gosec with appropriate exclusions for generated code.

### Common Security Findings

1. **HTTP Server Timeouts**: Fixed by adding proper timeout configurations
2. **File Path Traversal**: Fixed by adding path validation in config loaders
3. **Integer Overflow in Generated Code**: Excluded from security scans as these are false positives

## Build Issues

### Go Version Compatibility

If you encounter Go version compatibility issues:

1. Ensure you're using a consistent Go version:
   ```bash
   go version
   ```

2. If using golangci-lint, specify the Go toolchain:
   ```bash
   GOTOOLCHAIN=go1.22.5 golangci-lint run
   ```

3. Clean module cache if needed:
   ```bash
   go clean -modcache
   go mod tidy
   ```

## Development Tools

### Available Scripts

- `./scripts/security-check.sh` - Run security analysis with gosec
- `./scripts/validate-workflows.sh` - Validate GitHub Actions workflows

### Quality Checks

Run the full quality check suite:
```bash
# Build check
go build ./cmd/main.go

# Static analysis
go vet ./...

# Linting
GOTOOLCHAIN=go1.22.5 golangci-lint run

# Security scan
./scripts/security-check.sh

# Workflow validation
./scripts/validate-workflows.sh
```

## Getting Help

If you encounter issues not covered in this guide:

1. Check the project logs for detailed error messages
2. Verify your configuration files are correctly formatted
3. Ensure all dependencies are properly installed
4. Check GitHub Actions logs for workflow-specific issues

## Contributing

When contributing fixes for issues in this guide:

1. Update this documentation if you fix a new type of issue
2. Add appropriate tests to prevent regression
3. Update the relevant scripts in the `scripts/` directory
