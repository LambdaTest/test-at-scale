#!/bin/sh

chown -R nucleus:nucleus /home/nucleus 2>/dev/null
exec runuser -u nucleus -- /usr/local/bin/nucleus "$@"
