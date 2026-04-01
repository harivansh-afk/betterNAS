FROM golang:1.26-alpine AS build

WORKDIR /src
COPY apps/control-plane ./apps/control-plane

WORKDIR /src/apps/control-plane
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/control-plane ./cmd/control-plane

FROM alpine:3.21

WORKDIR /app
COPY --from=build /out/control-plane /usr/local/bin/control-plane

EXPOSE 3000
CMD ["control-plane"]
