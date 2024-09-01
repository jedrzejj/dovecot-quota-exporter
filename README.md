# Dovecot Quota Exporter

This project implements a standalone Prometheus exporter, which exposes all Dovecot usage data exported by Dovecot's quota_clone plugin as Prometheus metrics using an intermediate redis database.

## Configuration

### Redis

Install, configure and start a local redis instance.


### Exporter

Run the exporter as a docker image

```
docker run dovecot-quota-exporter:1.0
```

### Prometheus

prometheus.yml
```
global:
  scrape_interval: 1m

scrape_configs:
  - job_name: 'quota'
    static_configs:
      - targets: ['quota-exporter:9901']
```

web-config.yml
```
sys=$(hostname)
pwgen 24 1 > web-pass.$sys.txt
HASH=$(cat web-pass.$sys.txt | htpasswd -i -n -B mail-panel | cut -d: -f2)
cat <<END > web-config.$sys.yml
tls_server_config:
  cert_file: /etc/prometheus/server.crt
  key_file: /etc/prometheus/server.key

basic_auth_users:
  mail-panel: $HASH
END
```

```
openssl genrsa -out server.key 2048
chown nobody server.key
cat <<END > openssl.cnf
[v3_req]
subjectAltName=DNS:$(hostname -f)
END
openssl req -new -x509 -subj "/CN=$(hostname -f)" -days 3650 -key server.key -out server.crt -config openssl.cnf -extensions v3_req
rm openssl.cnf
```

### docker-compose

```
version: '3.8'

volumes:
  prometheus_data: {}
  redis_data: {}

networks:
  dove:
    driver: bridge
    driver_opts:
      com.docker.network.bridge.name: br-dove

services:
  redis:
    image: redis:7.4
    container_name: redis
    restart: unless-stopped
    volumes:
      - redis_data:/data
    command:
      - 'redis-server'
      - '--save'
      - '60'
      - '1'
      - '--loglevel'
      - 'warning'
    expose:
      - 6379
    ports:
      - 6379:6379
    networks:
      - dove

  quota-exporter:
    image: arum.man.poznan.pl:8443/dovecot-quota-exporter:latest
    container_name: quota-exporter
    restart: unless-stopped
    command:
      - '--redis=redis:6379'
    expose:
      - 9901
    networks:
      - dove

  prometheus:
    image: prom/prometheus:latest
    container_name: prometheus
    restart: unless-stopped
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - ./web-config.yml:/etc/prometheus/web-config.yml
      - ./server.key:/etc/prometheus/server.key
      - ./server.crt:/etc/prometheus/server.crt
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--web.config.file=/etc/prometheus/web-config.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
      - '--web.enable-lifecycle'
    expose:
      - 9090
    ports:
      - 9090:9090
    networks:
      - dove
```

### firewall

Make sure, that traffic on the `br-dove` bridge if forwarded.

## Dovecot

Enable quota_clone plugin and configure it to store usage data in a local redis database.

In `/etc/dovecot/conf.d/10-mail.conf` enable the plugin:
```
mail_plugins = $mail_plugins quota quota_clone
```

Add `/etc/dovecot/conf.d/91-quota.conf` and configure the plugin:
```
plugin {
  quota_clone_dict = redis:host=127.0.0.1:port=6379
}
```
