# Stage 1: build web dashboard (static SPA)
FROM node:22-alpine AS web
WORKDIR /web
COPY web/package.json web/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY web/ .
RUN pnpm run build:static

# Stage 2: build Go binary (with embedded dashboard assets)
FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /web/dist/client ./pkg/symphony/dashboard/static/
RUN CGO_ENABLED=0 go build -o /symphony ./cmd/symphony/main.go

# Stage 3: runtime image
FROM alpine:3.21
RUN apk add --no-cache openssh-client bash git ca-certificates
COPY --from=build /symphony /usr/local/bin/symphony
ENTRYPOINT ["symphony"]
