#!/bin/sh
if [ -S /var/run/docker.sock ]; then
    DOCKER_GID=$(stat -c '%g' /var/run/docker.sock 2>/dev/null)
    if [ -n "$DOCKER_GID" ] && [ "$DOCKER_GID" != "0" ]; then
        addgroup -g "$DOCKER_GID" docker 2>/dev/null || true
        addgroup appuser docker 2>/dev/null || true
    else
        addgroup appuser root 2>/dev/null || true
    fi
fi
exec su-exec appuser "$@"
