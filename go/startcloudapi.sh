#!/bin/bash
source ../setcloudenv.sh
export ENCRYPT_PAYLOADS=${1:-false}
go run ./api
