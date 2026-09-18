# syntax=docker/dockerfile:1.7

FROM node:14 AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN --mount=type=secret,id=npmrc,target=/root/.npmrc npm ci
COPY . .
RUN npm run build

FROM nginx:stable
WORKDIR /app
COPY ./docker_files/default.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist/ /app/
EXPOSE 80