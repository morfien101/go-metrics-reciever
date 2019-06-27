#! /bin/bash

# build project
rm -rf ./artifacts
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o artifacts/metrics-receiver
cat <<'EOF'>> ./artifacts/metric-receiver.conf
{
    "redis_server": {
        "redis_host": "172.17.0.1",
        "redis_port": "6379"
    },
    "web_server": {
        "listen_port": "8081"
    },
    "influx_server": {
        "host": "http://172.17.0.1",
        "port": "8086",
        "database": "telegraf"
    }
}
EOF

# build docker container

docker build -t morfien101/metrics-receiver:latest .