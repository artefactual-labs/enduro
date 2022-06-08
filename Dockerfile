# syntax = docker/dockerfile:1.3

ARG TARGET=enduro

FROM golang:1.17.9-alpine AS build-go
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .

FROM build-go AS build-enduro
RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	go build -o /out/enduro .

FROM build-go AS build-enduro-a3m-worker
RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	go build -o /out/enduro-a3m-worker ./cmd/enduro-a3m-worker

FROM alpine:3.15.4 AS base
ARG USER_ID=1000
ARG GROUP_ID=1000
RUN addgroup -g ${GROUP_ID} -S enduro
RUN adduser -u ${USER_ID} -S -D enduro enduro
USER enduro

FROM base AS enduro
COPY --from=build-enduro /out/enduro /home/enduro/bin/enduro
COPY --from=build-enduro /src/enduro.toml /home/enduro/.config/enduro.toml
CMD ["/home/enduro/bin/enduro", "--config", "/home/enduro/.config/enduro.toml"]

FROM base AS enduro-a3m-worker
COPY --from=build-enduro-a3m-worker /out/enduro-a3m-worker /home/enduro/bin/enduro-a3m-worker
COPY --from=build-enduro-a3m-worker /src/enduro.toml /home/enduro/.config/enduro.toml
CMD ["/home/enduro/bin/enduro-a3m-worker", "--config", "/home/enduro/.config/enduro.toml"]

FROM ${TARGET}
