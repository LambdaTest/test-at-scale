#!/bin/sh

chown -R nucleus:nucleus /coverage 2>/dev/null
exec runuser -u nucleus -- /home/nucleus/nucleus "$@"
