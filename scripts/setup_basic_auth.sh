#!/bin/bash

# Nginxコンテナ内でhtpasswdを作成
docker exec -i seicheese-nginx sh -c '
apk add --no-cache apache2-utils && \
htpasswd -bc /etc/nginx/htpasswd admin jumboebi && \
chown nginx:nginx /etc/nginx/htpasswd && \
chmod 600 /etc/nginx/htpasswd'

echo "Basic authentication has been set up successfully" 