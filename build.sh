#!/bin/bash
# Simple ZMK build script using Docker

set -e  # Exit on any error
set -o pipefail  # Catch errors in pipes

echo "Cleaning up old firmware"
rm -rf ./firmware/

DOCKER_IMAGE="zmkfirmware/zmk-build-arm:stable"

echo "Building Corne firmware..."
echo ""

# Build left side
echo "→ Building LEFT side..."
if ! docker run --rm \
    -u $(id -u):$(id -g) \
    -v "$(pwd):/workspace" \
    -w /workspace \
    -e ZEPHYR_BASE=/workspace/zephyr \
    "$DOCKER_IMAGE" \
    sh -c "
        set -e  # Exit on error inside docker
        [ ! -d .west ] && west init -l config/ || echo 'Workspace already initialized'
        west update
        west build -s zmk/app -b nice_nano_v2 -d build/left -- -DSHIELD='corne_left nice_view_adapter nice_view' -DZMK_CONFIG=/workspace/config -DCMAKE_PREFIX_PATH=/workspace/zephyr/share/zephyr-package/cmake
        mkdir -p firmware
        cp build/left/zephyr/zmk.uf2 firmware/corne_left.uf2
    "; then
    echo ""
    echo "✗ LEFT side build FAILED!"
    echo "  Check the error messages above."
    exit 1
fi

echo ""
echo "→ Building RIGHT side..."
if ! docker run --rm \
    -u $(id -u):$(id -g) \
    -v "$(pwd):/workspace" \
    -w /workspace \
    -e ZEPHYR_BASE=/workspace/zephyr \
    "$DOCKER_IMAGE" \
    sh -c "
        set -e  # Exit on error inside docker
        west build -s zmk/app -b nice_nano_v2 -d build/right -- -DSHIELD='corne_right nice_view_adapter nice_view' -DZMK_CONFIG=/workspace/config -DCMAKE_PREFIX_PATH=/workspace/zephyr/share/zephyr-package/cmake
        mkdir -p firmware
        cp build/right/zephyr/zmk.uf2 firmware/corne_right.uf2
    "; then
    echo ""
    echo "✗ RIGHT side build FAILED!"
    echo "  Check the error messages above."
    exit 1
fi

echo ""
echo "✓ Build complete!"
echo "  firmware/corne_left.uf2"
echo "  firmware/corne_right.uf2"
