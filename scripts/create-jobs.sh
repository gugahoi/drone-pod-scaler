#!/usr/bin/env bash

for i in $(seq 1 10); do
    echo "Creating new build: $i"
    drone build start momenton/coinfish-auth-go 1
done
