FROM node:lts-alpine

RUN apk add --no-cache \
    curl \
    go \
    bash \
    wget \
    libstdc++

RUN curl -fsSL https://raw.githubusercontent.com/versori/cli/main/install.sh | sh