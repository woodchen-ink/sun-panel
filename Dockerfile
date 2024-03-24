# build frontend
FROM node AS web_image

RUN npm install pnpm -g

WORKDIR /build

COPY package.json pnpm-lock.yaml /build/

RUN pnpm install

COPY . /build

RUN pnpm run build

# build backend
FROM golang:1.21-alpine3.18 AS server_image

WORKDIR /build

COPY service /build

RUN apk add --no-cache bash curl gcc git musl-dev \
    && go env -w GO111MODULE=on \
    && export PATH=$PATH:/go/bin \
    && go install -a -v github.com/go-bindata/go-bindata/...@latest \
    && go install -a -v github.com/elazarl/go-bindata-assetfs/...@latest \
    && go-bindata-assetfs -o=assets/bindata.go -pkg=assets assets/... \
    && go build -o sun-panel --ldflags="-X sun-panel/global.RUNCODE=release -X sun-panel/global.ISDOCKER=docker" main.go

# nginx and final setup
FROM nginx:alpine

WORKDIR /app

# Copy built assets
COPY --from=web_image /build/dist /app/web
COPY --from=server_image /build/sun-panel /app

# Setup Nginx
RUN rm /etc/nginx/conf.d/default.conf
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

# Setup start script
RUN echo -e "#!/bin/sh\n./sun-panel -port=3002 &\nnginx -g 'daemon off;'" > start.sh \
    && chmod +x start.sh

CMD ["./start.sh"]
