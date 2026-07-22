FROM node:22-bookworm-slim AS build

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY index.html postcss.config.js tailwind.config.js tsconfig.json vite.config.ts ./
COPY src ./src
# Vite exposes VITE_* vars (from .env or process env) as import.meta.env at build time.
# Defaults keep the deployed image auth-disabled; override via compose build args to enable.
ARG VITE_SIGNALOPS_AUTH_ENABLED=false
ARG VITE_SIGNALOPS_AUTH_ISSUER=https://auth.syncratic.co/realms/syncratic
ARG VITE_SIGNALOPS_AUTH_REALM=syncratic
ARG VITE_SIGNALOPS_AUTH_CLIENT_ID=signalops-web
ARG VITE_SIGNALOPS_AUTH_AUDIENCE=signalops-api
ARG VITE_SIGNALOPS_AUTH_IDLE_TIMEOUT_MINUTES=30
ARG VITE_SIGNALOPS_AUTH_RENEW_BEFORE_EXPIRY_SECONDS=300
ENV VITE_SIGNALOPS_AUTH_ENABLED=$VITE_SIGNALOPS_AUTH_ENABLED \
    VITE_SIGNALOPS_AUTH_ISSUER=$VITE_SIGNALOPS_AUTH_ISSUER \
    VITE_SIGNALOPS_AUTH_REALM=$VITE_SIGNALOPS_AUTH_REALM \
    VITE_SIGNALOPS_AUTH_CLIENT_ID=$VITE_SIGNALOPS_AUTH_CLIENT_ID \
    VITE_SIGNALOPS_AUTH_AUDIENCE=$VITE_SIGNALOPS_AUTH_AUDIENCE \
    VITE_SIGNALOPS_AUTH_IDLE_TIMEOUT_MINUTES=$VITE_SIGNALOPS_AUTH_IDLE_TIMEOUT_MINUTES \
    VITE_SIGNALOPS_AUTH_RENEW_BEFORE_EXPIRY_SECONDS=$VITE_SIGNALOPS_AUTH_RENEW_BEFORE_EXPIRY_SECONDS
RUN npm run build

FROM nginx:1.27-alpine AS web

COPY deploy/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 8080
