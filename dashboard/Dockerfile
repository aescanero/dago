# dashboard/Dockerfile — multi-stage build for the React 19 dashboard.
FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ARG VITE_API_URL=http://localhost:8080
ARG VITE_AUTH_URL=http://localhost:8081
ARG VITE_AUTH_CLIENT_ID=dago-dashboard
ARG VITE_AUTH_REDIRECT_URI=http://localhost/auth/callback
RUN npm run build

FROM nginx:1.27-alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
