FROM node:24-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .

ARG VITE_API_BASE_URL=/api/v1
ARG VITE_APP_TITLE=PromptOS
ARG VITE_ENABLE_PROMPT_API=true
ARG VITE_GITHUB_OAUTH_ENABLED=false

ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
ENV VITE_APP_TITLE=$VITE_APP_TITLE
ENV VITE_ENABLE_PROMPT_API=$VITE_ENABLE_PROMPT_API
ENV VITE_GITHUB_OAUTH_ENABLED=$VITE_GITHUB_OAUTH_ENABLED

RUN npm run build

FROM nginx:1.29-alpine3.22

RUN apk upgrade --no-cache

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80
