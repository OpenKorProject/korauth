FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o korauth ./cmd/korauth && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -o korauth-cli ./cmd/korauth-cli

# ---

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/korauth /app/korauth-cli .
COPY api/ api/

EXPOSE 8081
ENTRYPOINT ["./korauth"]
