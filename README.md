### Overview
A minimalistic daily logger. Create topics, log entries against them daily. 
If a topic has no entry for the day, it pushes an ntfy notification as a reminder.

Backend: Go + SQLite (modernc.org/sqlite, pure-Go, no CGO)
Frontend: plain HTML, no framework
Notifications: self-hosted ntfy over HTTP

### Running

    cp run.sh.example run.sh
    chmod +x run.sh

Edit run.sh with your NTFY_BASE, NTFY_USER, and NTFY_PASS, then:

    ./run.sh

Requires a running ntfy instance reachable at NTFY_BASE.

### Todo
- improve the UI
- host on a Pi 4B, accessible via LAN only (Cloudflare later)
- CI/CD pipeline on the Pi for auto builds and deploys