#!/bin/bash
source ../setcloudenv.sh
ENCRYPT_PAYLOADS=$1 go run worker/main.go
