# Stage 1: Build
# https://hub.docker.com/_/node/tags?name=alpine
FROM node:20-alpine AS builder

WORKDIR /build/web

# The build context is the repository root so this stage can copy the shared
# i18n dictionary (web/src/i18n/translations.ts imports ../../../shared/i18n).
# Only the whitelisted trees (web/ + shared/i18n/) are sent — see .dockerignore.
COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/ ./
COPY shared/ ../shared/
RUN npm run build

# Stage 2: Serve with nginx
# https://hub.docker.com/_/nginx/tags?name=alpine
FROM nginx:1.29-alpine

RUN apk upgrade --no-cache && apk add --no-cache ca-certificates tzdata

# Copy built frontend files
COPY --from=builder /build/web/dist /usr/share/nginx/html

# Copy custom nginx config
COPY web/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
