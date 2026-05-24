# Backend dev Dockerfile — runs air for hot reload. Production image (built
# from CI on a tagged release) will use a multi-stage build with a distroless
# runtime; that lives in Story 7.1.
FROM golang:1.25-alpine

WORKDIR /app

RUN apk add --no-cache git
RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

EXPOSE 8080

CMD ["air", "-c", ".air.toml"]
