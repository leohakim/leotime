FROM node:25-alpine AS web
WORKDIR /src
COPY package.json package-lock.json ./
COPY apps/web/package.json apps/web/package.json
RUN npm ci --workspace @leotime/web
COPY apps/web apps/web
WORKDIR /src/apps/web
RUN npm run build

FROM golang:1.26-alpine AS api
WORKDIR /src/apps/api
COPY apps/api/go.mod apps/api/go.sum* ./
RUN go mod download
COPY apps/api ./
RUN CGO_ENABLED=0 go build -o /out/leotime ./cmd/leotime

FROM alpine:3.22
RUN apk add --no-cache tzdata
RUN adduser -D -H -u 10001 leotime
WORKDIR /app
COPY --from=api /out/leotime /usr/local/bin/leotime
COPY --from=web /src/apps/web/dist /app/public
RUN mkdir -p /data && chown -R leotime:leotime /data /app
USER leotime
ENV LEOTIME_HTTP_ADDR=:8080
ENV LEOTIME_DB_PATH=/data/leotime.db
ENV LEOTIME_DOCUMENT_ROOT=/data/documents
ENV LEOTIME_STATIC_DIR=/app/public
EXPOSE 8080
VOLUME ["/data"]
CMD ["leotime"]
