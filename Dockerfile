#first stage - builder
FROM golang:1.21 AS builder

WORKDIR /app
COPY . /app

ARG GITLAB_ACCESS_TOKEN
RUN git config --global url."https://oauth2:${GITLAB_ACCESS_TOKEN}@git.larsa.io/".insteadOf "https://git.larsa.io/"

ENV GO111MODULE=on
RUN CGO_ENABLED=0 GOOS=linux go mod download
RUN CGO_ENABLED=0 GOOS=linux go build cmd/main.go
#second stage
FROM debian:bookworm-slim
WORKDIR /app

COPY --from=builder /app/main .
COPY .env .
RUN apt-get update && apt-get install -y \
    ca-certificates

EXPOSE 3232
CMD ["./main"]