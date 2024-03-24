# build frontend
FROM node AS web_image

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

RUN apk add --no-cache bash curl gcc git musl-dev

RUN go env -w GO111MODULE=on \
    && export PATH=$PATH:/go/bin \
    && go install -a -v github.com/go-bindata/go-bindata/...@latest \
    && go install -a -v github.com/elazarl/go-bindata-assetfs/...@latest \
    && go-bindata-assetfs -o=assets/bindata.go -pkg=assets assets/... \
    && go build -o sun-panel --ldflags="-X sun-panel/global.RUNCODE=release -X sun-panel/global.ISDOCKER=docker" main.go

# setup nginx
FROM nginx:alpine

WORKDIR /app

# Copy static files from web_image to nginx html directory
COPY --from=web_image /build/dist /app/web

# Copy backend binary from server_image to the current directory
COPY --from=server_image /build/sun-panel /app/sun-panel

# Custom Nginx Configuration
RUN rm /etc/nginx/conf.d/default.conf
COPY nginx.conf /etc/nginx/conf.d/app.conf

EXPOSE 80

# Ensure the sun-panel is executable
RUN chmod +x ./sun-panel

# Run sun-panel in the background and then start nginx
CMD ["sh", "-c", "./sun-panel & nginx -g 'daemon off;'"]
