FROM node:24-alpine AS build

WORKDIR /app

COPY src/frontend/package.json src/frontend/package-lock.json ./
RUN npm ci

COPY src/frontend/ ./
ARG VITE_MAX_BOT_URL=""
ENV VITE_MAX_BOT_URL=${VITE_MAX_BOT_URL}
RUN npm run build

FROM caddy:2.10-alpine

COPY --from=build /app/dist /srv
