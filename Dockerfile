# build frontend
FROM node AS web_image

# 华为源
# RUN npm config set registry https://repo.huaweicloud.com/repository/npm/

RUN npm install pnpm -g

WORKDIR /build

COPY ./package.json /build

COPY ./pnpm-lock.yaml /build

RUN pnpm install

COPY . /build

RUN pnpm run build

# build backend
FROM golang:1.21-alpine3.18 as server_image

WORKDIR /build

COPY ./service .

# 中国国内源
# RUN sed -i "s@dl-cdn.alpinelinux.org@mirrors.aliyun.com@g" /etc/apk/repositories \
#     && go env -w GOPROXY=https://goproxy.cn,direct

RUN apk add --no-cache bash curl gcc git musl-dev

RUN go env -w GO111MODULE=on \
    && export PATH=$PATH:/go/bin \
    && go install -a -v github.com/go-bindata/go-bindata/...@latest \
    && go install -a -v github.com/elazarl/go-bindata-assetfs/...@latest \
    && go-bindata-assetfs -o=assets/bindata.go -pkg=assets assets/... \
    && go build -o sun-panel --ldflags="-X sun-panel/global.RUNCODE=release -X sun-panel/global.ISDOCKER=docker" main.go

# setup nginx
FROM nginx:alpine

# Copy static files from web_image to nginx
COPY --from=web_image /build/dist /usr/share/nginx/html

# Copy backend binary from server_image to nginx container
COPY --from=server_image /build/sun-panel /usr/share/nginx/html/sun-panel

# Custom Nginx Configuration
RUN rm /etc/nginx/conf.d/default.conf
COPY nginx.conf /etc/nginx/conf.d

EXPOSE 80

# Run sun-panel in the background and then start nginx
CMD ["/bin/sh", "-c", "/usr/share/nginx/html/sun-panel & nginx -g 'daemon off;'"]
