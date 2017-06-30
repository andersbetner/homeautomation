#!/usr/bin/env bash

curl -X POST http://localhost:8080 -H "Content-Type: application/json" --data-binary "@test_sense.json"
