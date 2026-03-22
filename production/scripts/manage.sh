#!/bin/bash
#
# LocalFinance Management Script
#

case "$1" in
    start)
        echo "ð Starting LocalFinance services..."
        sudo systemctl start hisab-bot hisab-web
        echo "✅ Services started"
        ;;
    stop)
        echo "ð Stopping LocalFinance services..."
        sudo systemctl stop hisab-bot hisab-web
        echo "✅ Services stopped"
        ;;
    status)
        echo "ð LocalFinance Service Status:"
        systemctl status hisab-bot hisab-web --no-pager
        ;;
    restart)
        echo "ð Restarting LocalFinance services..."
        sudo systemctl restart hisab-bot hisab-web
        echo "✅ Services restarted"
        ;;
    logs)
        echo "ð Recent service logs:"
        journalctl -u hisab-bot -u hisab-web --no-pager -n 20
        ;;
    *)
        echo "Usage: $0 {start|stop|status|restart|logs}"
        exit 1
        ;;
esac
