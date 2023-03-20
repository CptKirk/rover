ARG TF_VERSION=1.4.0
# Build ui
FROM node:16-alpine as ui
WORKDIR /src
# Copy specific package files
COPY ./ui/package-lock.json ./
COPY ./ui/package.json ./
COPY ./ui/babel.config.js ./
# Set Progress, Config and install
RUN npm set progress=false && npm config set depth 0 && npm install
# Copy source
# Copy Specific Directories
COPY ./ui/public ./public
COPY ./ui/src ./src
# build (to dist folder)
RUN npm run build

# Build rover
FROM golang:1.20 AS builder
WORKDIR /src
# Copy full source
COPY ./go.* .
COPY ./*.go .
COPY --from=ui ./src/dist ./ui/dist
# Build rover
# RUN go get -d -v golang.org/x/net/html  
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o rover .

# Release stage
FROM hashicorp/terraform:$TF_VERSION as terraform-dep
FROM alpine:3.17
LABEL org.opencontainers.image.source="https://github.com/CptKirk/rover"

# Copy rover binary
COPY --from=builder /src/rover /bin/rover
COPY --from=terraform-dep /bin/terraform /bin/terraform
RUN chmod +x /bin/rover

RUN apk add --no-cache bash curl ca-certificates openssl

WORKDIR /src

ENTRYPOINT [ "/bin/rover" ]