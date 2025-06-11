#!/bin/bash

# Ensure script stops on any error
set -e

mkdir -p dist

# Navigate to shapeshifter directory
pushd game/mcp


# Remove existing binary in dist if it exists
if [ -f ../../dist/shapeshifter ]; then
    rm ../../dist/shapeshifter
fi

# Build shapeshifter directly to dist
go build -o ../../dist/shapeshifter cmd/main.go

# Return to previous directory
popd
