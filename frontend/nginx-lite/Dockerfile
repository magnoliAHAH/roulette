FROM ubuntu:24.04

COPY nginx* /tmp

RUN apt update && apt install -y\
    libbrotli1 \
    openssl

RUN chmod 755 /tmp/nginx*

RUN dpkg -i /tmp/*.deb

COPY nginx.conf /etc/nginx/nginx.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]