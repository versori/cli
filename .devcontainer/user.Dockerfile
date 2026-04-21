FROM node:lts-alpine

RUN apk add --no-cache \
    curl \
    bash \
    wget \
    libstdc++

RUN curl https://raw.githubusercontent.com/versori/cli/main/install.sh | sh