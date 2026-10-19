#!/bin/sh
# Root-owned deploy hook; a rejected config must not replace live workers.
set -eu
/usr/sbin/nginx -t
/usr/bin/systemctl reload nginx.service
