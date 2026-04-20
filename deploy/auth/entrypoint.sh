#!/bin/sh
chown appuser:appuser /app/uploads 2>/dev/null || true
exec su-exec appuser "$@"
