# syntax=docker/dockerfile:1
# Built from the repo root (see docker-compose.yml): one image with the API
# and the SPA, which the backend serves from STATIC_DIR.

FROM node:24-alpine AS web
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY backend/ ./
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/tracker ./cmd/tracker

FROM gcr.io/distroless/static-debian12:nonroot
ARG VERSION=dev
LABEL org.opencontainers.image.version=$VERSION
COPY --from=build /out/tracker /tracker
COPY --from=web /app/dist /web
ENV STATIC_DIR=/web
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/tracker"]
CMD ["serve"]
