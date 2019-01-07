#!/usr/bin/env bash

current_path=$(pwd)

cd $GOPATH/src/github.com/influxdata/influxdb/client/v2

# https://github.com/matryer/moq
moq -out=$current_path/influxdb_client_mock_test.go -pkg=monitoring . Client
