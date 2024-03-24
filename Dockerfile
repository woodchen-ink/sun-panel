# Build frontend
FROM node AS web_image

# Optionally set a mirror for npm for better performance
# RUN npm config set registry https://repo.huaweicloud.com/repository/npm/

RUN npm install pnpm -g

WORKDIR /build

COPY ./package.json /build
COPY ./pnpm-lock.yaml /build

RUN pnpm install

COPY . /build

RUN pnpm run build

# Build backend
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

# Setup the final image
FROM alpine

# Install Nginx
RUN apk add --no-cache nginx bash ca-certificates su-exec tzdata

# Setup directories
RUN mkdir -p /run/nginx

# Remove the default Nginx configuration
RUN rm /etc/nginx/conf.d/default.conf

# Copy the Nginx configuration from the build context
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Copy built assets from previous stages
COPY --from=web_image /build/dist /var/www/html
COPY --from=server_image /build/sun-panel /app/sun-panel

# Expose port 80 for Nginx
EXPOSE 80

# Add a script to start Nginx and sun-panel
COPY start.sh /start.sh
RUN chmod +x /start.sh

CMD ["/start.sh"]
