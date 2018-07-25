FROM scratch
ADD grafana-proxy grafana-proxy
ENTRYPOINT ["./grafana-proxy"]