# syntax=docker/dockerfile:1

FROM node:22-alpine AS frontend
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26.9-alpine AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# go:embed needs the frontend's build output in place before compiling.
COPY --from=frontend /src/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ztpanel ./cmd/ztpanel

FROM alpine:3.22
RUN apk add --no-cache ca-certificates wget \
 && addgroup -S ztpanel \
 && adduser -S ztpanel -G ztpanel

COPY --from=backend /out/ztpanel /usr/local/bin/ztpanel

# Holds ztpanel's own users/session key/organizations/activity log --
# not the ZeroTier controller's own data. Mount a volume here, or
# override ZTPANEL_DATA_DIR.
RUN mkdir -p /data && chown ztpanel:ztpanel /data
VOLUME ["/data"]
USER ztpanel
WORKDIR /data

ENV ZTPANEL_LISTEN=:8787 \
    ZTPANEL_DATA_DIR=/data
EXPOSE 8787
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://127.0.0.1:8787/healthz || exit 1

ENTRYPOINT ["ztpanel"]
