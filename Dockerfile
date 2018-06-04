FROM frolvlad/alpine-glibc:alpine-3.7

ARG BUILD_NUMBER

EXPOSE 9999/tcp 46656/tcp 46657/tcp 46658/tcp

WORKDIR /app/

ADD https://private.delegatecall.com/loom/linux/build-${BUILD_NUMBER}/loom /usr/bin/

RUN mkdir /app/contracts \
    && chmod +x /usr/bin/loom \
    && sync \
    && loom init \
    && echo 'QueryServerHost: "tcp://0.0.0.0:9999"' > /app/loom.yaml

CMD ["loom", "run"]
