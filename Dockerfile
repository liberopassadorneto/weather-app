FROM golang:1.20 as builder

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o weather-app .

FROM debian:stretch-slim

COPY --from=builder /app/weather-app /usr/local/bin/weather-app

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/weather-app"]
