#!/usr/bin/env bash
# Validates commit message against Conventional Commits spec
# Format: type(scope)?: description
# Types: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert

set -e

commit_msg_file="$1"
if [ -z "$commit_msg_file" ] || [ ! -f "$commit_msg_file" ]; then
  echo "Usage: $0 <path-to-commit-message-file>"
  exit 1
fi

# Read first line (subject)
subject=$(head -n 1 "$commit_msg_file")

# Conventional commit regex: type(scope)?: description
# Also allow Merge commits, Revert, and breaking change (type!:
regex='^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\([a-z0-9\-]+\))?!?: .{1,}$'
merge_regex='^Merge .+'
revert_regex='^Revert .+'

if echo "$subject" | grep -qE "$merge_regex|$revert_regex"; then
  exit 0
fi

if ! echo "$subject" | grep -qE "$regex"; then
  echo "Invalid commit message format."
  echo ""
  echo "Expected: type(scope)?: description"
  echo "Types: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert"
  echo ""
  echo "Examples:"
  echo "  feat(auth): add Google OAuth"
  echo "  fix(ui): fix button alignment"
  echo "  docs: update README"
  echo ""
  echo "Your message: $subject"
  exit 1
fi

exit 0
