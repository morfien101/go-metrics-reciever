#! /bin/bash


DOCKER_ACCOUNT=morfien101
PROJECT_NAME=metrics-receiver

# build project
rm -rf ./artifacts
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o artifacts/$PROJECT_NAME
cat <<'EOF'>> ./artifacts/metric-receiver.conf
{
    "auth_server": {
        "host": "http://172.17.0.1"
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

docker build -t $DOCKER_ACCOUNT/$PROJECT_NAME:latest .
docker tag $DOCKER_ACCOUNT/$PROJECT_NAME:latest $DOCKER_ACCOUNT/$PROJECT_NAME:$(docker run $DOCKER_ACCOUNT/$PROJECT_NAME:latest -v)