FROM golang:1.25-rc-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download


COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o tallyport

FROM alpine:3.18

WORKDIR /app

COPY --from=builder /app/tallyport .
COPY setting.yml .

RUN apk add --no-cache ca-certificates

EXPOSE 8080

CMD ["./tallyport", "-config-file", "setting.yml"]