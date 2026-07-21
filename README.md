### Overview
A minimalistic daily logger. Create topics, log entries against them daily. 
If a topic has no entry for the day, it pushes an ntfy notification as a reminder.

Backend: Go + SQLite (modernc.org/sqlite, pure-Go, no CGO)
Frontend: plain HTML, no framework
Notifications: self-hosted ntfy over HTTP

### Running locally

    cp run.sh.example run.sh
    chmod +x run.sh

Edit run.sh with your NTFY_BASE, NTFY_USER, and NTFY_PASS, then:

    ./run.sh

Requires a running ntfy instance reachable at NTFY_BASE.

### Deploying to the Pi (the original objective)

    ssh pi@<pi-ip> 'mkdir -p /opt/daily-logger'
    ssh pi@<pi-ip> 'sudo mkdir -p /var/lib/daily-logger && sudo chown pi:pi /var/lib/daily-logger'
    scp run.sh.example pi@<pi-ip>:/opt/daily-logger/run.sh
    scp daily-logger.service.example pi@<pi-ip>:/tmp/daily-logger.service
    ssh pi@<pi-ip> 'chmod +x /opt/daily-logger/run.sh'
    ssh pi@<pi-ip> 'sudo mv /tmp/daily-logger.service /etc/systemd/system/daily-logger.service && sudo systemctl daemon-reload && sudo systemctl enable --now daily-logger'

Edit /opt/daily-logger/run.sh on the Pi with real NTFY credentials before starting the service.

After this one-time setup, every push to main auto-builds and deploys via 
Gitea Actions (see deploy.yml), only the binary and templates/ are synced; 
run.sh and the persistent data directory (/var/lib/daily-logger) on the Pi 
are left untouched.

Requires SSH key auth already set up between the osdev-runner host and the Pi 
(no password prompt).

### Todo
- Cloudflare Tunnel for external access (currently LAN only)
- dockerfile and compose for self hosting