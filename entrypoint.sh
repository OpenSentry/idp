#!/bin/sh

go get -v -d ./...

update-ca-certificates

# This will exec the CMD from your Dockerfile, i.e. "npm start"
exec "$@"
