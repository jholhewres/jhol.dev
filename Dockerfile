FROM node:22-alpine AS web
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.24-alpine AS build
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
COPY --from=web /app/web/dist ./cmd/server/dist
RUN CGO_ENABLED=0 go build -o /bin/server ./cmd/server

FROM alpine:3.20
COPY --from=build /bin/server /server
COPY content/ /content/
EXPOSE 8123
CMD ["/server", "-content", "/content", "-port", "8123"]
