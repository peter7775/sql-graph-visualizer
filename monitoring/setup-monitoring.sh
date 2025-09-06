#!/bin/bash

# SQL Graph Visualizer - Monitoring Setup Script
# Sets up automated code monitoring via cron

set -e

PROJECT_DIR="/srv/http/sql-graph-visualizer"
SCRIPT_PATH="$PROJECT_DIR/monitoring/code-monitoring.sh"

echo "Setting up SQL Graph Visualizer code monitoring..."

# Make monitoring script executable
chmod +x "$SCRIPT_PATH"

# Check if required tools are available
echo "Checking dependencies..."

MISSING_DEPS=()

if ! command -v curl >/dev/null 2>&1; then
    MISSING_DEPS+=("curl")
fi

if ! command -v jq >/dev/null 2>&1; then
    MISSING_DEPS+=("jq")
fi

if [ ${#MISSING_DEPS[@]} -ne 0 ]; then
    echo "Missing dependencies: ${MISSING_DEPS[*]}"
    echo "Installing missing dependencies..."
    
    # Install on different systems
    if command -v pacman >/dev/null 2>&1; then
        # Arch/Manjaro
        sudo pacman -S --noconfirm "${MISSING_DEPS[@]}"
    elif command -v apt-get >/dev/null 2>&1; then
        # Ubuntu/Debian
        sudo apt-get update && sudo apt-get install -y "${MISSING_DEPS[@]}"
    elif command -v yum >/dev/null 2>&1; then
        # CentOS/RHEL
        sudo yum install -y "${MISSING_DEPS[@]}"
    else
        echo "Please install manually: ${MISSING_DEPS[*]}"
        exit 1
    fi
fi

# Create monitoring directories
mkdir -p "$PROJECT_DIR/monitoring/results"

# Setup cron jobs
echo "Setting up cron jobs..."

# Create crontab entries
CRON_TEMP=$(mktemp)

# Preserve existing crontab
crontab -l 2>/dev/null > "$CRON_TEMP" || true

# Remove any existing monitoring entries
grep -v "sql-graph-visualizer.*monitoring" "$CRON_TEMP" > "$CRON_TEMP.clean" 2>/dev/null || cp "$CRON_TEMP" "$CRON_TEMP.clean"

# Add new monitoring entries
cat >> "$CRON_TEMP.clean" << EOF

# SQL Graph Visualizer - Code Monitoring
# Daily monitoring at 9 AM
0 9 * * * cd $PROJECT_DIR && ./monitoring/code-monitoring.sh >> monitoring/cron.log 2>&1

# Weekly detailed monitoring on Mondays at 8 AM  
0 8 * * 1 cd $PROJECT_DIR && ./monitoring/code-monitoring.sh --detailed >> monitoring/cron.log 2>&1

EOF

# Install new crontab
crontab "$CRON_TEMP.clean"

# Cleanup
rm -f "$CRON_TEMP" "$CRON_TEMP.clean"

echo "✓ Cron jobs installed successfully"

# Create systemd service alternative (optional)
create_systemd_service() {
    cat > /tmp/sql-graph-monitor.service << EOF
[Unit]
Description=SQL Graph Visualizer Code Monitor
After=network.target

[Service]
Type=oneshot
User=$(whoami)
WorkingDirectory=$PROJECT_DIR
ExecStart=$SCRIPT_PATH
StandardOutput=append:$PROJECT_DIR/monitoring/service.log
StandardError=append:$PROJECT_DIR/monitoring/service.log

[Install]
WantedBy=multi-user.target
EOF

    cat > /tmp/sql-graph-monitor.timer << EOF
[Unit]
Description=Run SQL Graph Visualizer Code Monitor daily
Requires=sql-graph-monitor.service

[Timer]
OnCalendar=daily
Persistent=true

[Install]
WantedBy=timers.target
EOF

    echo "Systemd service files created in /tmp/"
    echo "To install systemd alternative:"
    echo "  sudo cp /tmp/sql-graph-monitor.* /etc/systemd/system/"
    echo "  sudo systemctl daemon-reload"
    echo "  sudo systemctl enable sql-graph-monitor.timer"
    echo "  sudo systemctl start sql-graph-monitor.timer"
}

# Test run
echo "Testing monitoring script..."
if "$SCRIPT_PATH" --test 2>/dev/null || "$SCRIPT_PATH"; then
    echo "✓ Monitoring script test successful"
else
    echo "⚠️ Monitoring script test failed, but continuing..."
fi

# Show current setup
echo ""
echo "=== Monitoring Setup Complete ==="
echo "Script location: $SCRIPT_PATH"
echo "Log location: $PROJECT_DIR/monitoring/monitoring-results.log"
echo "Results directory: $PROJECT_DIR/monitoring/results/"
echo ""
echo "Cron schedule:"
echo "  - Daily at 9 AM: Basic monitoring"
echo "  - Weekly on Monday at 8 AM: Detailed monitoring"
echo ""
echo "To manually run monitoring:"
echo "  cd $PROJECT_DIR && ./monitoring/code-monitoring.sh"
echo ""
echo "To view logs:"
echo "  tail -f $PROJECT_DIR/monitoring/monitoring-results.log"
echo ""

# Offer systemd alternative
read -p "Would you like to create systemd service files as well? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    create_systemd_service
fi

echo "Setup completed successfully!"
