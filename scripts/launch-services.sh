#!/bin/bash

# Ensure script stops on any error
set -e

# Kill existing processes
pkill -f shapeserver || true
pkill -f shapeshifter || true

# Launch shapeserver
pushd game/shapeserver/
nohup ./shapeserver > shapeserver.log 2>&1 &
popd

# Launch shapeshifter
pushd game/mcp
nohup ./shapeshifter --httpmode > shapeshifter.log 2>&1 &
popd

# Optional: Wait a moment and check if processes started
sleep 2

# Check if processes are running
pgrep -f shapeserver || echo "shapeserver failed to start"
pgrep -f shapeshifter || echo "shapeshifter failed to start"
