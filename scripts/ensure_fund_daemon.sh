#!/bin/bash
# Ensure auto_fund_daemon is running. Safe to call from cron repeatedly.
PIDFILE="/tmp/auto_fund_daemon.pid"
DAEMON="/usr/local/bin/auto_fund_daemon.sh"
LOGFILE="/tmp/auto_fund_daemon.log"

if [ -f "$PIDFILE" ] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
    # Daemon is running
    exit 0
fi

# Daemon is NOT running - restart it
echo "$(date '+%Y-%m-%d %H:%M:%S') Daemon not running, restarting..." >> "$LOGFILE"
nohup bash "$DAEMON" 120 5 20 >> /dev/null 2>&1 &
