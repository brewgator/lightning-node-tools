#!/bin/bash

# install-pre-commit-hook.sh - Install a pre-commit hook to run CI validation

set -e

HOOK_FILE=".git/hooks/pre-commit"

echo "🎣 Installing pre-commit hook..."

# Create the pre-commit hook
cat > "$HOOK_FILE" << 'EOF'
#!/bin/bash

# Pre-commit hook to run CI validation
echo "🔍 Running pre-commit validation..."

# Run the CI validation script
if ./scripts/validate-ci.sh > /dev/null 2>&1; then
    echo "✅ Pre-commit validation passed"
    exit 0
else
    echo "❌ Pre-commit validation failed"
    echo "Run './scripts/validate-ci.sh' to see details"
    echo "Fix issues before committing"
    exit 1
fi
EOF

# Make the hook executable
chmod +x "$HOOK_FILE"

echo "✅ Pre-commit hook installed successfully!"
echo ""
echo "The hook will run './scripts/validate-ci.sh' before each commit."
echo "To skip the hook for a specific commit, use: git commit --no-verify"
echo ""
echo "To remove the hook later, delete: $HOOK_FILE"