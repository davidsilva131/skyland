# Skyland API — imagen de producción (multi-stage, un build para api + worker)
FROM golang:1.27-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -o /out/worker ./cmd/worker

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/api /out/worker /app/
COPY migrations ./migrations

# API por defecto en :8080; el worker se lanza con command distinto (mismo build):
CMD ["/app/api"]
