#!/bin/bash
# uninstall.sh - This script runs on the target Ubuntu server.

set -e

# --- Configuration ---
SERVICE_NAME="belt-presense"
INSTALL_DIR="/opt/belt-presense"
# --- End Configuration ---

echo "--- Starting application uninstallation ---"

# Stop and disable the service
echo "Stopping and disabling $SERVICE_NAME service..."
sudo systemctl stop $SERVICE_NAME || true
sudo systemctl disable $SERVICE_NAME || true

# Remove the systemd file
echo "Removing systemd service file..."
sudo rm -f /etc/systemd/system/$SERVICE_NAME.service

# Reload systemd to apply changes
echo "Reloading systemd daemon..."
sudo systemctl daemon-reload

# Remove the installation directory
echo "Removing installation directory: $INSTALL_DIR..."
sudo rm -rf $INSTALL_DIR

echo "✅ Uninstallation complete."