FROM golang:1.26.1-alpine AS builder

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o cwa-exporter .

FROM gcr.io/distroless/static-debian12

COPY --from=builder /app/cwa-exporter /

EXPOSE 9100

ENTRYPOINT ["/cwa-exporter"]
