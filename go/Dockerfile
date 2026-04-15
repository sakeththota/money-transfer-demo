# Build stage for worker
FROM golang:1.23 AS builder
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOCACHE=/go/.cache
ARG BUILD_TARGET=worker
WORKDIR /go/app
COPY . .
RUN go build \
    -ldflags "-w -s" \
    -o /go/bin/app ./${BUILD_TARGET}

# Development stage with hot reload
FROM golang:1.23 AS dev
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOCACHE=/go/.cache
WORKDIR /go/app
RUN go install github.com/air-verse/air@v1.61.7
ENTRYPOINT [ "air" ]
CMD [ "-c", ".air.toml" ]

# API development stage
FROM golang:1.23 AS dev-api
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOCACHE=/go/.cache
WORKDIR /go/app
RUN go install github.com/air-verse/air@v1.61.7
ENTRYPOINT [ "air" ]
CMD [ "-c", ".air.api.toml" ]

# Codec server development stage
FROM golang:1.23 AS dev-codec
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOCACHE=/go/.cache
WORKDIR /go/app
RUN go install github.com/air-verse/air@v1.61.7
ENTRYPOINT [ "air" ]
CMD [ "-c", ".air.codec.toml" ]

# Production stage
FROM scratch AS production
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /go/bin/app /app
ENTRYPOINT [ "/app/app" ]
