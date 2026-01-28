#!/bin/bash

# Zen VPN Node Installation Script
# This script sets up a VPN node with sing-box, node agent, and Caddy fallback

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored message
print_msg() {
    local color=$1
    local msg=$2
    echo -e "${color}${msg}${NC}"
}

# Print header
print_header() {
    echo ""
    print_msg "$BLUE" "======================================"
    print_msg "$BLUE" "$1"
    print_msg "$BLUE" "======================================"
    echo ""
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        print_msg "$RED" "Error: This script must be run as root"
        exit 1
    fi
}

# Check prerequisites
check_prerequisites() {
    print_header "Checking Prerequisites"

    local missing=()

    # Check Docker
    if ! command -v docker &> /dev/null; then
        missing+=("docker")
    else
        print_msg "$GREEN" "✓ Docker is installed"
    fi

    # Check Docker Compose (v2)
    if docker compose version &> /dev/null; then
        print_msg "$GREEN" "✓ Docker Compose v2 is installed"
        COMPOSE_CMD="docker compose"
    elif docker-compose --version &> /dev/null; then
        print_msg "$GREEN" "✓ Docker Compose v1 is installed"
        COMPOSE_CMD="docker-compose"
    else
        missing+=("docker-compose")
    fi

    # Check curl
    if ! command -v curl &> /dev/null; then
        missing+=("curl")
    else
        print_msg "$GREEN" "✓ curl is installed"
    fi

    # Check openssl
    if ! command -v openssl &> /dev/null; then
        missing+=("openssl")
    else
        print_msg "$GREEN" "✓ openssl is installed"
    fi

    # Exit if missing prerequisites
    if [ ${#missing[@]} -ne 0 ]; then
        print_msg "$RED" ""
        print_msg "$RED" "Missing prerequisites: ${missing[*]}"
        print_msg "$YELLOW" ""
        print_msg "$YELLOW" "Install missing components:"

        if [[ " ${missing[*]} " =~ " docker " ]] || [[ " ${missing[*]} " =~ " docker-compose " ]]; then
            print_msg "$YELLOW" "  Docker: curl -fsSL https://get.docker.com | sh"
        fi

        if [[ " ${missing[*]} " =~ " curl " ]]; then
            print_msg "$YELLOW" "  curl: apt install curl (Debian/Ubuntu) or yum install curl (CentOS/RHEL)"
        fi

        if [[ " ${missing[*]} " =~ " openssl " ]]; then
            print_msg "$YELLOW" "  openssl: apt install openssl (Debian/Ubuntu) or yum install openssl (CentOS/RHEL)"
        fi

        exit 1
    fi

    # Check Docker daemon
    if ! docker info &> /dev/null; then
        print_msg "$RED" "Error: Docker daemon is not running"
        print_msg "$YELLOW" "Start Docker with: systemctl start docker"
        exit 1
    fi
    print_msg "$GREEN" "✓ Docker daemon is running"
}

# Generate secure random token
generate_token() {
    openssl rand -hex 32
}

# Create .env file
create_env_file() {
    print_header "Creating Configuration"

    if [ -f .env ]; then
        print_msg "$YELLOW" "Existing .env file found"
        read -p "Overwrite? (y/N): " confirm
        if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
            print_msg "$BLUE" "Keeping existing configuration"
            return
        fi
    fi

    # Generate API token
    API_TOKEN=$(generate_token)

    # Create .env file
    cat > .env << EOF
# Zen VPN Node Configuration
# Generated on $(date)

# API Token for node agent authentication
API_TOKEN=${API_TOKEN}

# Node agent listen address
LISTEN_ADDR=:9090

# Sing-box config file path
SINGBOX_CONFIG=/etc/sing-box/config.json

# Sing-box API endpoint
SINGBOX_API=http://127.0.0.1:10085
EOF

    chmod 600 .env
    print_msg "$GREEN" "✓ Configuration file created"
    print_msg "$YELLOW" ""
    print_msg "$YELLOW" "IMPORTANT: Save this API token for the admin panel:"
    print_msg "$GREEN" "  ${API_TOKEN}"
    echo ""
}

# Create default sing-box config
create_default_config() {
    print_header "Creating Default Sing-box Config"

    mkdir -p singbox

    if [ -f singbox/config.json ]; then
        print_msg "$YELLOW" "Existing sing-box config found, keeping it"
        return
    fi

    # Create a minimal working config (will be replaced by admin panel)
    cat > singbox/config.json << 'EOF'
{
  "log": {
    "level": "info",
    "timestamp": true
  },
  "experimental": {
    "v2ray_api": {
      "listen": "127.0.0.1:10085",
      "stats": {
        "enabled": true,
        "users": []
      }
    }
  },
  "inbounds": [],
  "outbounds": [
    {
      "type": "direct",
      "tag": "direct"
    }
  ]
}
EOF

    # Generate self-signed certificates for testing
    if [ ! -f singbox/cert.pem ] || [ ! -f singbox/key.pem ]; then
        print_msg "$BLUE" "Generating self-signed certificates..."
        openssl req -x509 -newkey rsa:2048 -keyout singbox/key.pem -out singbox/cert.pem \
            -days 365 -nodes -subj "/CN=localhost" 2>/dev/null
        print_msg "$GREEN" "✓ Self-signed certificates generated"
    fi

    print_msg "$GREEN" "✓ Default sing-box config created"
    print_msg "$YELLOW" "Note: Configure inbounds via the admin panel"
}

# Pull Docker images
pull_images() {
    print_header "Pulling Docker Images"

    print_msg "$BLUE" "Pulling sing-box image..."
    docker pull ghcr.io/sagernet/sing-box:latest

    print_msg "$BLUE" "Pulling Caddy image..."
    docker pull caddy:2-alpine

    print_msg "$GREEN" "✓ All images pulled successfully"
}

# Build and start services
start_services() {
    print_header "Starting Services"

    print_msg "$BLUE" "Building node agent..."
    $COMPOSE_CMD build agent

    print_msg "$BLUE" "Starting all services..."
    $COMPOSE_CMD up -d

    # Wait for services to start
    sleep 5

    # Check service status
    print_msg "$BLUE" "Checking service status..."

    if docker ps --format '{{.Names}}' | grep -q "node-agent"; then
        print_msg "$GREEN" "✓ Node agent is running"
    else
        print_msg "$RED" "✗ Node agent failed to start"
    fi

    if docker ps --format '{{.Names}}' | grep -q "singbox"; then
        print_msg "$GREEN" "✓ Sing-box is running"
    else
        print_msg "$YELLOW" "! Sing-box may need configuration (check logs)"
    fi

    if docker ps --format '{{.Names}}' | grep -q "caddy-fallback"; then
        print_msg "$GREEN" "✓ Caddy fallback is running"
    else
        print_msg "$RED" "✗ Caddy fallback failed to start"
    fi
}

# Configure firewall
configure_firewall() {
    print_header "Firewall Configuration"

    print_msg "$YELLOW" "The following ports need to be open:"
    echo "  - 443/tcp  : VPN traffic (REALITY/WebSocket)"
    echo "  - 443/udp  : VPN traffic (Hysteria2)"
    echo "  - 9090/tcp : Node agent API (restrict to admin panel IP)"
    echo ""

    # Check for common firewall tools
    if command -v ufw &> /dev/null; then
        print_msg "$BLUE" "UFW detected. Suggested commands:"
        echo "  ufw allow 443/tcp"
        echo "  ufw allow 443/udp"
        echo "  ufw allow from ADMIN_PANEL_IP to any port 9090"
    elif command -v firewall-cmd &> /dev/null; then
        print_msg "$BLUE" "firewalld detected. Suggested commands:"
        echo "  firewall-cmd --permanent --add-port=443/tcp"
        echo "  firewall-cmd --permanent --add-port=443/udp"
        echo "  firewall-cmd --permanent --add-rich-rule='rule family=\"ipv4\" source address=\"ADMIN_PANEL_IP\" port port=\"9090\" protocol=\"tcp\" accept'"
        echo "  firewall-cmd --reload"
    elif command -v iptables &> /dev/null; then
        print_msg "$BLUE" "iptables detected. Suggested commands:"
        echo "  iptables -A INPUT -p tcp --dport 443 -j ACCEPT"
        echo "  iptables -A INPUT -p udp --dport 443 -j ACCEPT"
        echo "  iptables -A INPUT -p tcp -s ADMIN_PANEL_IP --dport 9090 -j ACCEPT"
    fi

    echo ""
    print_msg "$YELLOW" "Note: Replace ADMIN_PANEL_IP with your admin panel's IP address"
}

# Print connection info
print_info() {
    print_header "Installation Complete!"

    # Get public IP
    PUBLIC_IP=$(curl -s -4 ifconfig.me 2>/dev/null || echo "UNABLE_TO_DETECT")

    # Read API token from .env
    if [ -f .env ]; then
        source .env
    fi

    echo "Node Information:"
    echo "================="
    echo ""
    print_msg "$GREEN" "Public IP: ${PUBLIC_IP}"
    print_msg "$GREEN" "Agent Port: 9090"
    print_msg "$GREEN" "API Token: ${API_TOKEN}"
    echo ""
    echo "Add this node to your Zen VPN Panel with:"
    echo "  - Address: ${PUBLIC_IP}"
    echo "  - Port: 9090"
    echo "  - Token: ${API_TOKEN}"
    echo ""
    echo "Useful Commands:"
    echo "================"
    echo "  View logs:     ${COMPOSE_CMD} logs -f"
    echo "  Stop services: ${COMPOSE_CMD} down"
    echo "  Start services: ${COMPOSE_CMD} up -d"
    echo "  Restart:       ${COMPOSE_CMD} restart"
    echo ""
    print_msg "$YELLOW" "Next Steps:"
    echo "  1. Configure firewall rules (see above)"
    echo "  2. Add this node in the Zen VPN Panel"
    echo "  3. Create inbounds with proper SNI/domain"
    echo "  4. Ensure domain DNS points to this server's IP"
    echo ""
}

# Main installation
main() {
    print_header "Zen VPN Node Installer"

    # Change to script directory
    cd "$(dirname "$0")"

    check_root
    check_prerequisites
    create_env_file
    create_default_config
    pull_images
    start_services
    configure_firewall
    print_info
}

# Run main function
main "$@"
