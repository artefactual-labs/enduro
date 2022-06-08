# syntax = docker/dockerfile:1.4

FROM node:18 AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/app/.npm \
    npm set cache /app/.npm && npm install-clean
COPY . .
RUN npm run build
CMD npm run dev

FROM nginx:alpine
WORKDIR /usr/share/nginx/html
RUN rm -rf ./*
COPY --from=builder /app/dist .
COPY <<EOF /etc/nginx/templates/default.conf.template
upstream backend {
    least_conn;
    server \${ENDURO_API_ADDRESS};
}
server {
    listen 80;
    root /usr/share/nginx/html;
    absolute_redirect off;
    location /api/ {
        proxy_pass http://backend/;
        proxy_redirect / /api/;
    }
    location / {
        try_files \$uri \$uri/ /index.html;
    }
}
EOF
