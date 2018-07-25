FROM iron/base
WORKDIR /app
ADD grafana-proxy /app/main
ENTRYPOINT ["./main"]