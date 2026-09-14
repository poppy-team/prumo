# prumo-agentd operations

The local daemon (`prumo agent serve`) is single-instance per project root
(PID lock at `.prumo/runtime/harness/agentd.pid`, stale takeover when the
pid is dead), refuses a second server, and stops via `prumo agent stop`
(SIGTERM; stale locks are cleared with a message).

Timelines rotate at 2000 JSONL lines (newest 1000 kept); checkpoints prune
to the newest 5 per run. Records, knowledge and evidence are never pruned —
provenance outlives trimming. `agent gc [--keep N] [--max-age-days N]`
collects aged artifacts of runs that keep no checkpoints (orphaned partial
runs) and reports counts; live runs are untouched.

Example systemd unit (user scope):

```ini
[Unit]
Description=Prumo Harness daemon (%h project)
After=network-online.target

[Service]
Type=simple
WorkingDirectory=%h/Documentos/Projetos/prumo
ExecStart=%h/go/bin/prumo agent serve --path %h/Documentos/Projetos/prumo
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
```
