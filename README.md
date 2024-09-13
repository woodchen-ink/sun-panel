# SUN-PANEL

``` yml
services:
  sun-panel:
    container_name: sun-panel
    image: woodchen/sun-panel
    ports:
      - "5002:3002"
    volumes:
      - ./conf:/app/conf
      - ./conf/uploads:/app/uploads
      - ./conf/database:/app/database
      - ./conf/custom:/app/web/custom
    restart: always
    networks:
      - 1panel-network
networks:
  1panel-network:
    external: true
```
直接按照docker-compose.yml文件来部署就行
