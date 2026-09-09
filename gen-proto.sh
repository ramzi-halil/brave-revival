#!/bin/sh

set -e

PATH=$(go env GOBIN):$PATH protoc --go_opt=module=example.com/brave-revival -I=src/proto --go_out . src/proto/*.proto
