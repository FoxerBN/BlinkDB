FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/miniredis ./cmd/server

FROM alpine:3.22

RUN adduser -D -H -s /sbin/nologin miniredis

WORKDIR /app
COPY --from=build /out/miniredis /app/miniredis

USER miniredis
EXPOSE 6379

ENTRYPOINT ["/app/miniredis"]
