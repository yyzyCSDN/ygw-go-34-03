FROM golang:1.23

WORKDIR /app

COPY go.mod go.sum ./
COPY vendor ./vendor
COPY . .

ENV GOPROXY=off \
    GOSUMDB=off \
    CGO_ENABLED=0

RUN go build -mod=vendor -o /out/configd ./cmd/configd

EXPOSE 8080

CMD ["/out/configd", "-addr", "0.0.0.0:8080"]
