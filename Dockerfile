FROM node:22-alpine AS dashboard
WORKDIR /app/web
COPY web/package.json ./package.json
RUN npm install
COPY web/ ./
RUN npm run build

FROM golang:1.23-alpine AS go-build
WORKDIR /src
COPY go.mod ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/loadforge ./cmd/loadforge

FROM alpine:3.21
RUN apk --no-cache add ca-certificates && addgroup -S loadforge && adduser -S -G loadforge loadforge
WORKDIR /app
COPY --from=go-build /out/loadforge /usr/local/bin/loadforge
COPY --from=dashboard /app/web/dist ./web/dist
USER loadforge
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/loadforge", "serve", "--addr", ":8080"]
