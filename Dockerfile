FROM golang:1.27-alpine AS build

WORKDIR /build
COPY src/go.mod src/go.sum ./
RUN go mod download

COPY src/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/person-service ./cmd/server
RUN go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@v4.20.1

FROM alpine:3.22

RUN apk add --no-cache ca-certificates && addgroup -S app && adduser -S -G app app
COPY --from=build /out/person-service /usr/local/bin/person-service
COPY --from=build /go/bin/migrate /usr/local/bin/migrate
COPY src/migrations /migrations
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint
RUN chmod +x /usr/local/bin/docker-entrypoint

USER app
EXPOSE 8080
ENTRYPOINT ["docker-entrypoint"]
CMD ["person-service"]
