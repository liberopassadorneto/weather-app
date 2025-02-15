FROM golang:1.20 as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o weather-app .

FROM debian:stretch-slim

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/weather-app /usr/local/bin/weather-app

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/weather-app"]
