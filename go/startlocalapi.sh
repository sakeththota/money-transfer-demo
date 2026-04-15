#!/bin/bash
export ENCRYPT_PAYLOADS=${1:-false}
go run ./api
