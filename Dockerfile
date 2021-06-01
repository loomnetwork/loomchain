FROM ubuntu:latest

ARG BUILD_NUMBER
ARG ARCH

EXPOSE 26656/tcp 6656/tcp 46657/tcp 46658/tcp

WORKDIR /app/

ADD https://downloads.loomx.io/loom/linux${ARCH}/build-${BUILD_NUMBER}/loom /usr/bin/

RUN mkdir /app/contracts \
    && chmod +x /usr/bin/loom \
    && sync \
    && loom init

CMD ["loom", "run"]
