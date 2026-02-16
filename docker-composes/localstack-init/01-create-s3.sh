#!/usr/bin/env sh
set -e
awslocal s3 mb s3://shared-local-dev 2>/dev/null || true
