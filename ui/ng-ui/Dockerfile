# syntax=docker/dockerfile:1.7

FROM node:14 AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN --mount=type=secret,id=npmrc,target=/root/.npmrc npm ci
COPY . .
RUN --mount=type=secret,id=environment_prod,target=/run/secrets/environment.prod.ts \
    if [ -f /run/secrets/environment.prod.ts ]; then \
      cp /run/secrets/environment.prod.ts src/environments/environment.prod.ts; \
    fi; \
    test -f src/environments/environment.prod.ts || \
      (echo "src/environments/environment.prod.ts must be provided as the environment_prod build secret" >&2; exit 1); \
    npm run build; \
    rm -f src/environments/environment.prod.ts

FROM nginx:stable
WORKDIR /app
COPY ./docker_files/default.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist/ /app/
EXPOSE 80