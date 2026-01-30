#!/bin/bash

# Load environment variables from .env file (skip comments, empty lines, and incomplete values)
if [ -f .env ]; then
    while IFS='=' read -r key value; do
        # Skip comments, empty lines, and lines with ...
        if [[ ! $key =~ ^# ]] && [[ -n $key ]] && [[ ! $value =~ \.\.\. ]]; then
            export "$key"="$value"
        fi
    done < <(grep -v '^#' .env | grep -v '^\s*$')
fi

# Start the server
go run .
