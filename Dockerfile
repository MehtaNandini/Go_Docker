# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.22.5
FROM golang:${GO_VERSION}-alpine AS build

RUN apk add --no-cache build-base git

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .

# Ensure module graph and sums are up to date
RUN go mod tidy

# Build static binary where possible
ENV CGO_ENABLED=1
RUN go build -trimpath -ldflags "-s -w" -o /out/todo ./cmd/server

FROM alpine:3.20
RUN adduser -S -D -H appuser
USER appuser

WORKDIR /app
COPY --from=build /out/todo /app/todo

ENV PORT=8080
EXPOSE 8080

# Create data dir for sqlite database
RUN mkdir -p /app/data
VOLUME ["/app/data"]

ENTRYPOINT ["/app/todo"]



