#!/bin/sh
set -eu

AUTH_URL="${RESTREAM_AUTH_URL:-http://state-api:8080/api/v1/restream/auth}"
sed "s|\${RESTREAM_AUTH_URL}|${AUTH_URL}|g" /etc/mediamtx/mediamtx.yml.template > /tmp/mediamtx.yml
exec /usr/local/bin/mediamtx /tmp/mediamtx.yml
