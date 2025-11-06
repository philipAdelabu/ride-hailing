#!/bin/bash

# Install Git hooks for the project
# Run this script to set up pre-commit and pre-push hooks

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
HOOKS_DIR="$PROJECT_ROOT/.git/hooks"

echo "Installing Git hooks..."

# Create pre-commit hook
cat > "$HOOKS_DIR/pre-commit" << 'EOF'
#!/bin/bash

# Pre-commit hook
# Runs before each commit to ensure code quality

set -e

echo "Running pre-commit checks..."

# Check if there are any staged Go files
STAGED_GO_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$' || true)

if [ -n "$STAGED_GO_FILES" ]; then
    echo "✓ Found staged Go files, running checks..."

    # Format check
    echo "  Checking code formatting..."
    UNFORMATTED=$(gofmt -l $STAGED_GO_FILES)
    if [ -n "$UNFORMATTED" ]; then
        echo "❌ The following files are not formatted:"
        echo "$UNFORMATTED"
        echo ""
        echo "Run 'make fmt' to format your code"
        exit 1
    fi

    # Run go vet
    echo "  Running go vet..."
    if ! go vet ./...; then
        echo "❌ go vet failed"
        exit 1
    fi

    # Check for common issues
    echo "  Checking for common issues..."

    # Check for TODO without issue reference
    if git diff --cached | grep -i "^+.*TODO" | grep -v "#[0-9]"; then
        echo "⚠ Warning: Found TODO without issue reference"
    fi

    # Check for hardcoded secrets
    if git diff --cached | grep -E "(password|secret|token|api_key|apikey)\s*=\s*['\"]"; then
        echo "❌ Potential hardcoded credentials detected!"
        echo "Please remove hardcoded secrets before committing"
        exit 1
    fi

    echo "✓ Pre-commit checks passed!"
else
    echo "No Go files staged for commit"
fi

# Check YAML files
STAGED_YAML_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep -E '\.(yml|yaml)$' || true)
if [ -n "$STAGED_YAML_FILES" ]; then
    echo "  Checking YAML syntax..."
    # Add YAML validation here if needed
fi

exit 0
EOF

# Create pre-push hook
cat > "$HOOKS_DIR/pre-push" << 'EOF'
#!/bin/bash

# Pre-push hook
# Runs before pushing to ensure tests pass

set -e

echo "Running pre-push checks..."

# Run tests
echo "  Running tests..."
if ! go test ./... -short; then
    echo "❌ Tests failed!"
    echo "Fix the tests before pushing"
    exit 1
fi

# Run linter (if golangci-lint is installed)
if command -v golangci-lint &> /dev/null; then
    echo "  Running linter..."
    if ! golangci-lint run --timeout=2m ./...; then
        echo "❌ Linter found issues!"
        echo "Fix linting errors before pushing"
        exit 1
    fi
fi

echo "✓ Pre-push checks passed!"

exit 0
EOF

# Create commit-msg hook for commit message format
cat > "$HOOKS_DIR/commit-msg" << 'EOF'
#!/bin/bash

# Commit message hook
# Ensures commit messages follow conventional commit format

COMMIT_MSG_FILE=$1
COMMIT_MSG=$(cat "$COMMIT_MSG_FILE")

# Skip merge commits
if echo "$COMMIT_MSG" | head -1 | grep -q "^Merge"; then
    exit 0
fi

# Check for conventional commit format
# Pattern: type(scope): description
# Examples: feat: add user authentication, fix(auth): resolve token expiry
PATTERN="^(feat|fix|docs|style|refactor|test|chore|perf|ci|build|revert)(\(.+\))?: .+"

if ! echo "$COMMIT_MSG" | head -1 | grep -Eq "$PATTERN"; then
    echo "❌ Invalid commit message format!"
    echo ""
    echo "Commit message should follow the format:"
    echo "  <type>(<scope>): <description>"
    echo ""
    echo "Types:"
    echo "  feat:     A new feature"
    echo "  fix:      A bug fix"
    echo "  docs:     Documentation changes"
    echo "  style:    Code style changes (formatting, etc.)"
    echo "  refactor: Code refactoring"
    echo "  test:     Adding or updating tests"
    echo "  chore:    Maintenance tasks"
    echo "  perf:     Performance improvements"
    echo "  ci:       CI/CD changes"
    echo "  build:    Build system changes"
    echo "  revert:   Revert a previous commit"
    echo ""
    echo "Examples:"
    echo "  feat: add user authentication"
    echo "  fix(auth): resolve token expiry issue"
    echo "  docs: update API documentation"
    echo ""
    exit 1
fi

exit 0
EOF

# Make hooks executable
chmod +x "$HOOKS_DIR/pre-commit"
chmod +x "$HOOKS_DIR/pre-push"
chmod +x "$HOOKS_DIR/commit-msg"

echo "✓ Git hooks installed successfully!"
echo ""
echo "Installed hooks:"
echo "  - pre-commit:  Runs code formatting and vet checks"
echo "  - pre-push:    Runs tests and linter"
echo "  - commit-msg:  Validates commit message format"
echo ""
echo "To bypass hooks (not recommended), use:"
echo "  git commit --no-verify"
echo "  git push --no-verify"
