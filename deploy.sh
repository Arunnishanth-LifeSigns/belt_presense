#!/bin/bash
# deploy.sh - To be run from the WSL/Ubuntu terminal.

# Exit immediately if a command exits with a non-zero status.
set -e

# --- Configuration ---
BASTION_HOST="13.127.85.4"
TARGET_HOST="10.15.133.150"
# IMPORTANT: Update this path to the WSL location of your PEM key
LOCAL_PEM_KEY="~/.ssh/ls-central-dev-common.pem"
BASTION_PEM_KEY="~/ls-central-dev-common.pem"
SERVICE_NAME="belt-presense"
SSH_USER="ubuntu"
# --- End Configuration ---

echo "--- Starting package-based deployment to $TARGET_HOST via Bastion $BASTION_HOST ---"

# =================================================================
## 1. Build and Package the Application
# =================================================================
echo "--- Building and packaging the application... ---"

BINARY_NAME="${SERVICE_NAME}-svc"
export GOOS=linux
export GOARCH=arm64
export CC=aarch64-linux-gnu-gcc
# Build natively inside Linux - no cross-compilation needed!
CGO_ENABLED=1 go build -o ./${BINARY_NAME} ./cmd/main.go

# Create a temporary package directory
PACKAGE_DIR="./package"
rm -rf "$PACKAGE_DIR"
mkdir -p "$PACKAGE_DIR"

# --- NEW STEP: Fix line endings before packaging ---
echo "--- Converting shell scripts to Unix format... ---"
dos2unix install.sh
dos2unix uninstall.sh

# Copy all necessary files into the package directory
cp ./${BINARY_NAME} "$PACKAGE_DIR/"
# Copy .env.prod and rename it to .env for the package
cp ./.env.prod "${PACKAGE_DIR}/.env"
cp ./install.sh "$PACKAGE_DIR/"
# cp ./uninstall.sh "$PACKAGE_DIR/" # Uncomment if you have this file

# Create a compressed tarball
PACKAGE_NAME="package.tar.gz"
tar -czvf "$PACKAGE_NAME" -C "$PACKAGE_DIR" .

echo "✅ Package created: $PACKAGE_NAME"
echo "----------------------------------------------------"


# =================================================================
## 2. Transfer and Execute the Package
# =================================================================
TEMP_REMOTE_PATH="/tmp"
REMOTE_PACKAGE_PATH="${TEMP_REMOTE_PATH}/${PACKAGE_NAME}"

# A. Copy package from Local -> Bastion
echo "--> (Step 1/3) Copying package to Bastion host..."
scp -i "$LOCAL_PEM_KEY" "$PACKAGE_NAME" "${SSH_USER}@${BASTION_HOST}:${TEMP_REMOTE_PATH}/"

# B. Move package from Bastion -> Target
echo "--> (Step 2/3) Moving package to Target host..."
SCP_COMMAND="scp -i ${BASTION_PEM_KEY} -o StrictHostKeyChecking=no ${REMOTE_PACKAGE_PATH} ${SSH_USER}@${TARGET_HOST}:~/"
ssh -i "$LOCAL_PEM_KEY" "${SSH_USER}@${BASTION_HOST}" "$SCP_COMMAND"

# C. Unpack and run the installer on Target
echo "--> (Step 3/3) Running installer on Target host..."
INSTALL_COMMAND=$(cat <<'EOF'
set -e
echo '--- Unpacking and installing on target ---'
INSTALL_DIR="~/install_package"
mkdir -p "$INSTALL_DIR"
tar -xzvf ~/package.tar.gz -C "$INSTALL_DIR"
sudo "$INSTALL_DIR"/install.sh
echo '--- Cleaning up installation files ---'
rm ~/package.tar.gz
rm -rf "$INSTALL_DIR"
EOF
)

REMOTE_EXEC_COMMAND="ssh -i ${BASTION_PEM_KEY} -o StrictHostKeyChecking=no ${SSH_USER}@${TARGET_HOST} '${INSTALL_COMMAND}'"
ssh -i "$LOCAL_PEM_KEY" "${SSH_USER}@${BASTION_HOST}" "$REMOTE_EXEC_COMMAND"

echo "🚀 Deployment to $TARGET_HOST completed successfully!"

# Clean up local build files
rm "$PACKAGE_NAME"
rm "$BINARY_NAME"
rm -rf "$PACKAGE_DIR"