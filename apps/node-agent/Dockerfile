FROM golang:1.26-alpine AS build

WORKDIR /src
COPY apps/node-agent ./apps/node-agent

WORKDIR /src/apps/node-agent
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/node-agent ./cmd/node-agent

FROM alpine:3.21

WORKDIR /app
COPY --from=build /out/node-agent /usr/local/bin/node-agent

EXPOSE 8090
CMD ["node-agent"]
