FROM scratch
WORKDIR /app
ADD grafana-proxy /app/main
ENTRYPOINT ["./main"]