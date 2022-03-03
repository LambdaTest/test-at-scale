#!/bin/sh

chown -R nucleus:nucleus /workspace-cache 2>/dev/null
exec runuser -u nucleus -- /usr/local/bin/nucleus "$@"
