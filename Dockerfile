FROM frolvlad/alpine-glibc:alpine-3.7

ARG BUILD_NUMBER

EXPOSE 26656/tcp 6656/tcp 46657/tcp 46658/tcp

WORKDIR /app/

ADD https://downloads.loomx.io/loom/linux/build-${BUILD_NUMBER}/loom /usr/bin/

RUN mkdir /app/contracts \
    && chmod +x /usr/bin/loom \
    && sync \
    && loom init

CMD ["loom", "run"]
