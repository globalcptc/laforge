FROM postgres:15-alpine

COPY backup.sh /usr/local/bin/backup
COPY healthcheck.sh /usr/local/bin/backup-healthcheck

RUN mkdir -p /backups && \
    chown postgres:postgres /backups && \
    chmod 0755 /usr/local/bin/backup /usr/local/bin/backup-healthcheck

USER postgres

ENTRYPOINT ["/usr/local/bin/backup"]
