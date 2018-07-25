#!/bin/bash

GOOS=linux GOARCH=amd64 go build -o grafana-proxy ./src/
#go build -o grafana-proxy ./src/