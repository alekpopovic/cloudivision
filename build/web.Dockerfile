FROM node:24-alpine AS build
WORKDIR /app
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM nginx:1.29-alpine
COPY --from=build /app/dist/cloudivision-web/browser /usr/share/nginx/html
COPY build/web-nginx.conf /etc/nginx/conf.d/default.conf
COPY build/web-entrypoint.sh /docker-entrypoint.d/99-cloudivision-config.sh
RUN chmod +x /docker-entrypoint.d/99-cloudivision-config.sh \
  && chown -R nginx:nginx /usr/share/nginx/html /var/cache/nginx /var/run /var/log/nginx
USER nginx
