# Stage 1: build web dashboard (static SPA)
FROM node:22-alpine AS web
WORKDIR /web
COPY web/package.json web/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY web/ .
RUN pnpm run build:static

# Stage 2: build Go binaries (with embedded dashboard assets)
FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /web/dist/client ./pkg/symphony/dashboard/static/
RUN CGO_ENABLED=0 go build -o /symphony ./cmd/symphony/main.go
RUN CGO_ENABLED=0 go build -o /anclax ./cmd/main.go

# Stage 3: symphony standalone runtime image
FROM alpine:3.21 AS symphony
RUN apk add --no-cache openssh-client bash git ca-certificates
COPY --from=build /symphony /usr/local/bin/symphony
ENTRYPOINT ["symphony"]

# Stage 4: anclax full-stack runtime image (API + DB + Symphony manager)
FROM alpine:3.21 AS anclax
RUN apk add --no-cache openssh-client bash git ca-certificates
COPY --from=build /anclax /usr/local/bin/anclax
ENTRYPOINT ["anclax"]
