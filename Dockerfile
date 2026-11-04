FROM node:20-alpine AS webapp-builder

ARG NPM_REGISTRY=https://registry.npmmirror.com
ENV COREPACK_NPM_REGISTRY=${NPM_REGISTRY} \
    npm_config_registry=${NPM_REGISTRY}

WORKDIR /app
RUN corepack enable && corepack prepare pnpm@10.32.1 --activate
COPY webapp/package.json ./webapp/
COPY webapp/pnpm-lock.yaml ./webapp/
COPY webapp_mobile/package.json ./webapp_mobile/
COPY webapp_mobile/pnpm-lock.yaml ./webapp_mobile/
RUN set -eu; \
    for attempt in 1 2 3; do \
      cd /app/webapp && pnpm install --frozen-lockfile --registry="${NPM_REGISTRY}" && break; \
      if [ "$attempt" -eq 3 ]; then exit 1; fi; \
      sleep 2; \
    done
RUN set -eu; \
    for attempt in 1 2 3; do \
      cd /app/webapp_mobile && pnpm install --frozen-lockfile --registry="${NPM_REGISTRY}" && break; \
      if [ "$attempt" -eq 3 ]; then exit 1; fi; \
      sleep 2; \
    done

COPY webapp ./webapp
COPY webapp_mobile ./webapp_mobile
RUN cd webapp && pnpm run build
RUN cd webapp_mobile && pnpm run build

FROM golang:1.24.0-alpine AS backend-builder

ARG GOPROXY_PRIMARY=https://proxy.golang.org,direct
ARG GOPROXY_FALLBACK=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY_PRIMARY}

WORKDIR /src
COPY go.mod go.sum ./
RUN set -eu; \
    if go env -w GOPROXY="${GOPROXY_PRIMARY}" && go mod download; then \
      exit 0; \
    fi; \
    go env -w GOPROXY="${GOPROXY_FALLBACK}"; \
    for attempt in 1 2 3; do \
      go mod download && exit 0; \
      if [ "$attempt" -eq 3 ]; then exit 1; fi; \
      sleep 2; \
    done

COPY . .
ARG TARGETOS
ARG TARGETARCH
ARG BUILD_TIME
ARG GIT_COMMIT
ARG VERSION
RUN CGO_ENABLED=0 \
    GOOS=${TARGETOS:-$(go env GOOS)} \
    GOARCH=${TARGETARCH:-$(go env GOARCH)} \
    go build -ldflags="-s -w -X 'github.com/likaia/nginxpulse/internal/version.Version=${VERSION}' -X 'github.com/likaia/nginxpulse/internal/version.BuildTime=${BUILD_TIME}' -X 'github.com/likaia/nginxpulse/internal/version.GitCommit=${GIT_COMMIT}'" \
    -o /out/nginxpulse ./cmd/nginxpulse/main.go

FROM nginx:1.27-alpine AS runtime

WORKDIR /app
ARG BUILD_TIME
ARG GIT_COMMIT
ARG VERSION
RUN apk add --no-cache su-exec \
    postgresql \
    postgresql-client \
    && addgroup -S nginxpulse \
    && adduser -S nginxpulse -G nginxpulse \
    && mkdir -p /tmp \
    && chmod 1777 /tmp

COPY --from=backend-builder /out/nginxpulse /app/nginxpulse
COPY entrypoint.sh /app/entrypoint.sh
COPY scripts/embedded_postgres.sh /app/embedded_postgres.sh
COPY scripts/postgres-app-role.sql /app/postgres-app-role.sql
COPY scripts/postgres-app-owner.sql /app/postgres-app-owner.sql
COPY docs/external_ips.txt /app/assets/external_ips.txt
COPY --from=webapp-builder /app/webapp/dist /usr/share/nginx/html
COPY --from=webapp-builder /app/webapp_mobile/dist /usr/share/nginx/html/m
COPY configs/nginx_frontend.conf /etc/nginx/conf.d/default.conf
RUN mkdir -p /app/var/nginxpulse_data /app/var/pgdata /app/assets /app/configs \
    && chown -R nginxpulse:nginxpulse /app \
    && chmod +x /app/entrypoint.sh

LABEL org.opencontainers.image.title="nginxpulse" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${GIT_COMMIT}" \
      org.opencontainers.image.created="${BUILD_TIME}"
EXPOSE 8088 8089
ENTRYPOINT ["/app/entrypoint.sh"]
