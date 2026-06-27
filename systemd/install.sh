#!/bin/bash
set -e

# korauth systemd installation script
# Run as root: sudo bash systemd/install.sh

if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root (use: sudo bash systemd/install.sh)"
   exit 1
fi

echo "Installing korauth with systemd..."

# Directories
BASE_DIR="/opt/openkor/korauth"
BIN_DIR="$BASE_DIR/bin"
CONFIG_DIR="$BASE_DIR/config"
KEYS_DIR="$CONFIG_DIR/keys"

# Create directories
echo "Creating directories..."
mkdir -p "$BIN_DIR"
mkdir -p "$CONFIG_DIR"
mkdir -p "$KEYS_DIR"

# Create korauth user if it doesn't exist
if ! id "korauth" &>/dev/null; then
    echo "Creating korauth system user..."
    useradd --system --home "$BASE_DIR" --shell /bin/false korauth
fi

# Build binaries
echo "Building korauth binaries..."
go build -o korauth ./cmd/korauth
go build -o korauth-cli ./cmd/korauth-cli

# Install binaries
echo "Installing binaries to $BIN_DIR..."
install -o korauth -g korauth -m 0755 korauth "$BIN_DIR/"
install -o root -g root -m 0755 korauth-cli "$BIN_DIR/"
rm korauth korauth-cli

# Install systemd service
echo "Installing systemd service..."
install -o root -g root -m 0644 systemd/korauth.service /etc/systemd/system/

# Copy environment template
echo "Creating environment file template..."
if [[ ! -f "$CONFIG_DIR/korauth.env" ]]; then
    install -o korauth -g korauth -m 0600 systemd/korauth.env.example "$CONFIG_DIR/korauth.env"
    echo "⚠️  Please edit: $CONFIG_DIR/korauth.env"
else
    echo "ℹ️  $CONFIG_DIR/korauth.env already exists (skipped)"
fi

# Set up directory permissions
echo "Setting up directory permissions..."
chown -R korauth:korauth "$BASE_DIR"
chmod 755 "$BASE_DIR"
chmod 755 "$BIN_DIR"
chmod 755 "$CONFIG_DIR"
chmod 700 "$KEYS_DIR"

# Reload systemd daemon
echo "Reloading systemd daemon..."
systemctl daemon-reload

echo ""
echo "✓ Installation complete!"
echo ""
echo "Next steps:"
echo "1. Generate RSA keys:"
echo "   sudo openssl genrsa -out $KEYS_DIR/jwt-private.pem 4096"
echo "   sudo openssl rsa -in $KEYS_DIR/jwt-private.pem -pubout -out $KEYS_DIR/jwt-public.pem"
echo "   sudo chown korauth:korauth $KEYS_DIR/*.pem"
echo "   sudo chmod 600 $KEYS_DIR/*.pem"
echo ""
echo "2. Configure environment:"
echo "   sudo nano $CONFIG_DIR/korauth.env"
echo "   Edit DATABASE_URL, REDIS_URL, and JWT key paths"
echo ""
echo "3. Start the service:"
echo "   sudo systemctl start korauth"
echo ""
echo "4. Enable on boot:"
echo "   sudo systemctl enable korauth"
echo ""
echo "5. Check status:"
echo "   sudo systemctl status korauth"
echo "   sudo journalctl -u korauth -f  # View logs"
echo ""
echo "Admin utilities:"
echo "   Reset admin password:"
echo "   $BIN_DIR/korauth-cli reset-admin-password <tenant-id> <new-password>"
echo ""
