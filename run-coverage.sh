#!/bin/bash

go test -race -coverprofile=coverage.txt -covermode=atomic ./...

go tool cover -html=coverage.txt -o coverage.html

firefox coverage.html
