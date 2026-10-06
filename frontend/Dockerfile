FROM nginx:1.27-alpine

RUN rm -rf /usr/share/nginx/html && mkdir -p /usr/share/nginx/html

COPY ./site /usr/share/nginx/html

EXPOSE 80

CMD ["nginx","-g","daemon off;"]
