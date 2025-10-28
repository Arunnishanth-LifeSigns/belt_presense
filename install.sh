#!/bin/bash
# install.sh - This script runs on the target Ubuntu server.

set -e

# --- Configuration ---
SERVICE_NAME="belt-presense"
BINARY_NAME="belt-presense-svc"
INSTALL_DIR="/opt/belt-presense"
# --- End Configuration ---

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

echo "--- Starting application installation ---"

if systemctl is-active --quiet $SERVICE_NAME; then
    sudo systemctl stop $SERVICE_NAME
fi
if systemctl is-enabled --quiet $SERVICE_NAME; then
    sudo systemctl disable $SERVICE_NAME
fi

echo "Creating installation directory at $INSTALL_DIR..."
sudo mkdir -p $INSTALL_DIR

echo "Copying application files from $SCRIPT_DIR..."
sudo cp "$SCRIPT_DIR"/$BINARY_NAME $INSTALL_DIR/
sudo cp "$SCRIPT_DIR"/.env $INSTALL_DIR/
sudo chmod +x $INSTALL_DIR/$BINARY_NAME

echo "Setting permissions for the ubuntu user..."
sudo chown -R ubuntu:ubuntu $INSTALL_DIR

# --- NEW DEBUGGING STEP: Verify the directory contents ---
echo "--- Verifying contents of $INSTALL_DIR ---"
sudo ls -la $INSTALL_DIR

# Create the systemd service file
echo "Creating systemd service file..."
sudo tee /etc/systemd/system/$SERVICE_NAME.service > /dev/null <<EOF
[Unit]
Description=Belt Presense Data Processing Service
After=network.target

[Service]
User=ubuntu
Group=ubuntu
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/$BINARY_NAME
Restart=always
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

echo "Reloading systemd and starting the service..."
sudo systemctl daemon-reload
sudo systemctl enable $SERVICE_NAME.service
sudo systemctl start $SERVICE_NAME

echo ""
echo "✅ Installation complete. Service '$SERVICE_NAME' is active."
sudo systemctl status $SERVICE_NAME --no-pager