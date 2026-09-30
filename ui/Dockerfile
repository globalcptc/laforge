# Build the static bundle, then serve it. Nothing from this build lands on
# the host -- the compiled assets live only inside the image.

# --- build ---
FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
# Vite inlines this at build time (see ui/src/api/client.ts), so it is a
# build arg, not a runtime env var. Compose passes it per environment from
# PUBLIC_BASE_URL; rebuild the image if that address changes.
ARG VITE_API_BASE_URL=http://localhost:8080
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
RUN npm run build

# --- serve ---
FROM caddy:2-alpine
COPY --from=build /app/dist /srv
COPY Caddyfile /etc/caddy/Caddyfile
EXPOSE 80
