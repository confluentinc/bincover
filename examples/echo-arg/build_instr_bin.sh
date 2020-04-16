#!/usr/bin/env bash
go test . -tags testbincover -coverpkg=./... -c -o instr_bin -ldflags="-X github.com/confluentinc/bincover/examples/echo-arg.isTest=true"
