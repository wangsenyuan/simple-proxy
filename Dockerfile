FROM scratch
ADD grafana-proxy main
ENTRYPOINT ["./main"]