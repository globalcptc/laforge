FROM node:14 as builder
WORKDIR /app
COPY . .
RUN npm ci && npm run build

FROM nginx:stable
WORKDIR /app
COPY ./docker_files/default.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist/ /app/
EXPOSE 80 443