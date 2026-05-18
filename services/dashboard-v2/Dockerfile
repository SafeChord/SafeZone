# Stage 1: Build
FROM node:22-alpine AS builder

WORKDIR /build
COPY services/dashboard-v2/package.json services/dashboard-v2/package-lock.json ./
RUN npm ci

COPY services/dashboard-v2/ ./
RUN npm run build

# Stage 2: Serve
FROM nginx:1.27-alpine

COPY --from=builder /build/dist /usr/share/nginx/html
COPY utils/geo_data/boundaries/geo_city.json /usr/share/nginx/html/geo/
COPY utils/geo_data/boundaries/regions/ /usr/share/nginx/html/geo/regions/
COPY utils/geo_data/boundaries/accessory/city_code.json /usr/share/nginx/html/geo/
COPY utils/geo_data/boundaries/accessory/city_center.json /usr/share/nginx/html/geo/
COPY services/dashboard-v2/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
