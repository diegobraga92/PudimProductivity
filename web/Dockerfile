# Stage 1: Build
FROM node:20-alpine AS builder  # https://hub.docker.com/_/node/tags?name=alpine

WORKDIR /build

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN npm run build

# Stage 2: Serve with nginx
FROM nginx:1.27-alpine  # https://hub.docker.com/_/nginx/tags?name=alpine

RUN apk add --no-cache ca-certificates tzdata

# Copy built frontend files
COPY --from=builder /build/dist /usr/share/nginx/html

# Copy custom nginx config
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
