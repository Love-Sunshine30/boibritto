# Stage 1: Build
FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o boibritto .

# Stage 2: Run
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/boibritto .
COPY --from=builder /build/templates ./templates
COPY --from=builder /build/static ./static
COPY --from=builder /build/sw.js ./sw.js
COPY --from=builder /build/manifest.json ./manifest.json

EXPOSE 15000

ENTRYPOINT ["./boibritto"]