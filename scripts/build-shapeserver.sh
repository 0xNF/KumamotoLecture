#!/bin/bash

# Ensure script stops on any error
set -e

mkdir -p dist

# Navigate to shapeserver directory
pushd game/shapeserver/


# Remove existing binary in dist if it exists
if [ -f ../../dist/shapeserver ]; then
    rm ../../dist/shapeserver
fi

# Build shapeserver directly to dist
go build -o ../../dist/shapeserver

# Return to previous directory
popd
