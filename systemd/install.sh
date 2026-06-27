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
INSTALL_DIR="/opt/korauth"
CONFIG_DIR="/etc/korauth"
KEYS_DIR="/etc/korauth/keys"
LOG_DIR="/var/log/korauth"
LIB_DIR="/var/lib/korauth"

# Create directories
echo "Creating directories..."
mkdir -p "$INSTALL_DIR"
mkdir -p "$CONFIG_DIR"
mkdir -p "$KEYS_DIR"
mkdir -p "$LOG_DIR"
mkdir -p "$LIB_DIR"

# Create korauth user if it doesn't exist
if ! id "korauth" &>/dev/null; then
    echo "Creating korauth system user..."
    useradd --system --home /var/lib/korauth --shell /bin/false korauth
fi

# Build binaries
echo "Building korauth binaries..."
go build -o korauth ./cmd/korauth
go build -o korauth-cli ./cmd/korauth-cli

# Install binaries
echo "Installing binaries to $INSTALL_DIR..."
install -o korauth -g korauth -m 0755 korauth "$INSTALL_DIR/"
install -o root -g root -m 0755 korauth-cli "$INSTALL_DIR/"
rm korauth korauth-cli

# Install systemd service
echo "Installing systemd service..."
install -o root -g root -m 0644 systemd/korauth.service /etc/systemd/system/

# Copy environment template
echo "Creating environment file template..."
if [[ ! -f "$CONFIG_DIR/korauth.env" ]]; then
    install -o root -g root -m 0600 systemd/korauth.env.example "$CONFIG_DIR/korauth.env"
    echo "⚠️  Please edit: $CONFIG_DIR/korauth.env"
else
    echo "ℹ️  $CONFIG_DIR/korauth.env already exists (skipped)"
fi

# Set up keys directory
echo "Setting up keys directory..."
chown -R korauth:korauth "$KEYS_DIR"
chmod 700 "$KEYS_DIR"

# Set up logs directory
echo "Setting up logs directory..."
chown -R korauth:korauth "$LOG_DIR"
chmod 755 "$LOG_DIR"

# Set up lib directory
echo "Setting up library directory..."
chown -R korauth:korauth "$LIB_DIR"
chmod 755 "$LIB_DIR"

# Reload systemd daemon
echo "Reloading systemd daemon..."
systemctl daemon-reload

echo ""
echo "✓ Installation complete!"
echo ""
echo "Next steps:"
echo "1. Generate RSA keys:"
echo "   sudo mkdir -p $KEYS_DIR"
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
