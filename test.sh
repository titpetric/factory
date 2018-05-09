#!/bin/bash
set -e
go test ./prof* -v -cover
go test ./sonyflake* -v -cover
go test ./resputil -v -cover -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html