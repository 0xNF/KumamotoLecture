#!/bin/bash

# Ensure script stops on any error
set -e

# Kill existing processes
pkill -f shapeserver || true
pkill -f shapeshifter || true


pushd dist
ln -s ~/server.crt server.crt
ln -s ~/server.key server.key
# Launch shapeserver
nohup ./shapeserver > shapeserver.log 2>&1 &
# Launch shapeshifter
nohup ./shapeshifter --httpmode > shapeshifter.log 2>&1 &
popd

# Optional: Wait a moment and check if processes started
sleep 2

# Check if processes are running
pgrep -f shapeserver || echo "shapeserver failed to start"
pgrep -f shapeshifter || echo "shapeshifter failed to start"