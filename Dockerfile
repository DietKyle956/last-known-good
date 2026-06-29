FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /usr/local/bin/agent ./cmd/agent

FROM alpine:3.21
RUN apk add --no-cache ca-certificates git
COPY --from=build /usr/local/bin/agent /usr/local/bin/agent
ENTRYPOINT ["agent"]
