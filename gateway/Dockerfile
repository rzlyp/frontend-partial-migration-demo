FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY proxy-common.conf /etc/nginx/proxy-common.conf
COPY migrated-paths.conf /etc/nginx/migrated-paths.conf
COPY migrated-routes.json /etc/nginx/migrated-routes.json
