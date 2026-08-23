FROM nginxinc/nginx-unprivileged:1.29-alpine
COPY --chown=nginx:nginx index.html /usr/share/nginx/html/index.html
USER nginx
EXPOSE 8080
