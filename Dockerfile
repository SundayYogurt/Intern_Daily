# Build the React frontend
FROM node:22-bookworm-slim AS web
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Build the Go server (templates are embedded into the binary)
FROM golang:1.25-bookworm AS server
WORKDIR /app/server
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
RUN CGO_ENABLED=0 go build -ldflags '-s -w' -o /out/internship-journal .

# Runtime: Chromium for PDF export + Thai fonts
FROM debian:bookworm-slim
RUN apt-get update \
 && apt-get install -y --no-install-recommends chromium fonts-thai-tlwg fonts-noto-core ca-certificates tzdata \
 && rm -rf /var/lib/apt/lists/*
WORKDIR /app/server
COPY --from=server /out/internship-journal ./internship-journal
COPY --from=web /app/web/dist /app/web/dist
ENV WEB_DIST=/app/web/dist \
    PDF_BROWSER=/usr/bin/chromium \
    TZ=Asia/Bangkok
EXPOSE 8080
# Render (and most PaaS) pass the port in $PORT
CMD ["sh", "-c", "ADDR=0.0.0.0:${PORT:-8080} exec ./internship-journal"]
