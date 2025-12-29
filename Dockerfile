FROM alpine:latest

RUN apk add --no-cache nginx dnsmasq ca-certificates

COPY tube /usr/local/bin/tube

RUN chmod +x /usr/local/bin/tube

ENTRYPOINT ["/usr/local/bin/tube"]
