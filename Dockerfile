FROM silverstagtech/distroless:latest

LABEL maintainer="Randy Coburn <morfien101@gmail.com>"

COPY ./artifacts/* /
ENV METRIC_RECEIVER_CONFIG=/metric-receiver.conf

ENTRYPOINT [ "/metrics-receiver" ]