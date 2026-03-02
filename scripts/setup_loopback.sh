#!/bin/bash
# Setup loopback aliases for macOS Docker Desktop networking
# These aliases allow the host to access containers by their internal IPs
# Run with: sudo ./setup_loopback.sh

set -e

echo "Setting up loopback aliases for 0Chain local network..."

# Miners
echo "Configuring Miners (198.18.0.71-74)..."
sudo ifconfig lo0 alias 198.18.0.71
sudo ifconfig lo0 alias 198.18.0.72
sudo ifconfig lo0 alias 198.18.0.73
sudo ifconfig lo0 alias 198.18.0.74

# Sharders
echo "Configuring Sharders (198.18.0.81-82)..."
sudo ifconfig lo0 alias 198.18.0.81
sudo ifconfig lo0 alias 198.18.0.82

# 0dns
echo "Configuring 0dns (198.18.0.100)..."
sudo ifconfig lo0 alias 198.18.0.100

# Blobbers (7-12)
echo "Configuring Blobbers (198.18.0.97-99, 198.18.0.110-112)..."
sudo ifconfig lo0 alias 198.18.0.97
sudo ifconfig lo0 alias 198.18.0.98
sudo ifconfig lo0 alias 198.18.0.99
sudo ifconfig lo0 alias 198.18.0.110
sudo ifconfig lo0 alias 198.18.0.111
sudo ifconfig lo0 alias 198.18.0.112

# 0box
echo "Configuring 0box (198.18.9.2)..."
sudo ifconfig lo0 alias 198.18.9.2

echo ""
echo "Loopback aliases configured successfully!"
echo ""
echo "Note: These aliases are lost on reboot. Run this script after each restart."
echo ""
echo "You can verify with: ifconfig lo0 | grep 198.18"
