# GitHub Actions CI/CD Setup

This project includes comprehensive GitHub Actions workflows for continuous integration, security scanning, and automated releases.

## Workflows Overview

### 1. CI Pipeline (`ci.yml`)

Runs automatically on:
- Push to `main` or `develop` branches
- Pull requests to `main` or `develop` branches

**Jobs:**
- **Test**: Runs on multiple Go versions (1.21, 1.22)
  - Install dependencies
  - Run unit tests
  - Generate coverage reports
  - Upload coverage to Codecov
  - Run linting with `golangci-lint`
  - Generate API documentation
  - Build all components
  - Run integration tests

- **Docker Build**: 
  - Build Docker images
  - Test container functionality with health checks
  - Runs after successful tests

- **Security**:
  - Run Gosec security scanner
  - Run govulncheck for vulnerability detection

- **Code Quality**:
  - Run goreportcard analysis
  - Run staticcheck static analysis

### 2. Release Pipeline (`release.yml`)

Triggered by:
- Creating version tags (e.g., `v1.0.0`)

**Features:**
- Build and test all components
- Build and push Docker images to registry
- Create GitHub releases with binaries
- Generate release notes automatically

**Required Secrets:**
- `DOCKER_USERNAME`: Docker Hub username
- `DOCKER_PASSWORD`: Docker Hub password/token
- `GITHUB_TOKEN`: Automatically provided by GitHub

### 3. Dependency Updates (`dependency-update.yml`)

Runs automatically:
- Weekly on Sundays at 2 AM UTC
- Can be manually triggered

**Features:**
- Updates all Go dependencies
- Runs tests to ensure compatibility
- Creates pull request with changes
- Automated dependency management

### 4. Security Scanning (`codeql.yml`)

Runs automatically:
- On push/PR to main branches
- Weekly security scans on Mondays

**Features:**
- CodeQL security analysis
- Vulnerability detection
- Security alerts in GitHub Security tab

## Setup Instructions

### 1. Enable Workflows

1. Ensure workflows are in `.github/workflows/` directory
2. Push to repository - workflows activate automatically
3. Check Actions tab in GitHub repository

### 2. Configure Secrets

For releases to work, add these secrets in repository settings:

```bash
# Repository Settings > Secrets and Variables > Actions
DOCKER_USERNAME=your-dockerhub-username
DOCKER_PASSWORD=your-dockerhub-token
```

### 3. Branch Protection

Recommended branch protection rules for `main`:

- Require status checks to pass
- Require branches to be up to date
- Require review from code owners
- Include administrators in restrictions

### 4. Badge Integration

Add these badges to your main README.md:

```markdown
![CI](https://github.com/your-username/telemetry-pipeline/workflows/CI/badge.svg)
![Release](https://github.com/your-username/telemetry-pipeline/workflows/Release/badge.svg)
![Security](https://github.com/your-username/telemetry-pipeline/workflows/CodeQL/badge.svg)
[![codecov](https://codecov.io/gh/your-username/telemetry-pipeline/branch/main/graph/badge.svg)](https://codecov.io/gh/your-username/telemetry-pipeline)
```

## Workflow Customization

### Modifying Go Versions

Edit the matrix in `ci.yml`:
```yaml
strategy:
  matrix:
    go-version: [1.21, 1.22, 1.23]  # Add/remove versions
```

### Adding New Tests

Add test commands to the CI workflow:
```yaml
- name: Run new test suite
  run: make test-new-feature
```

### Custom Docker Registry

To use a different registry, update `release.yml`:
```yaml
- name: Log in to Custom Registry  
  uses: docker/login-action@v3
  with:
    registry: your-registry.com
    username: ${{ secrets.REGISTRY_USERNAME }}
    password: ${{ secrets.REGISTRY_PASSWORD }}
```

## Monitoring and Maintenance

### 1. Check Workflow Status

- Visit repository Actions tab
- Monitor for failures
- Review security alerts

### 2. Update Dependencies

- Dependency update PRs created automatically
- Review and merge weekly updates
- Monitor for breaking changes

### 3. Release Management

- Create tags for releases: `git tag v1.0.0 && git push --tags`
- Releases created automatically
- Docker images published to registry

### 4. Security Monitoring

- Check Security tab for vulnerabilities  
- Review CodeQL alerts
- Update dependencies for security fixes

## Troubleshooting

### Common Issues

1. **Lint Failures**: `golangci-lint` automatically installed
2. **Test Failures**: Review test output in Actions logs
3. **Docker Build Issues**: Check Dockerfile syntax
4. **Secret Errors**: Verify secrets are configured correctly

### Debug Actions Locally

Use `act` to test workflows locally:
```bash
# Install act
brew install act  # or other installation method

# Run CI workflow
act -j test

# Run with secrets
act -j test -s GITHUB_TOKEN=your_token
```

This setup provides comprehensive CI/CD with automated testing, security scanning, and release management for the telemetry pipeline project.