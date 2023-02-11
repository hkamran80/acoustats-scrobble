# syntax=docker/dockerfile:1

## Build
FROM golang:1.19-alpine AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN CGO_ENABLED=0 go build -o /acoustats-scrobble

## Deploy
FROM gcr.io/distroless/base-debian11

WORKDIR /

COPY --from=build /acoustats-scrobble /acoustats-scrobble

USER nonroot:nonroot

ENTRYPOINT ["/acoustats-scrobble"]