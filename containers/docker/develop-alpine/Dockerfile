FROM alpine:3.7

RUN \
  apk add --update go git make gcc musl-dev linux-headers ca-certificates && \
  git clone --depth 1 https://github.com/genchain/go-genchain && \
  (cd go-genchain && make ggen) && \
  cp go-genchain/build/bin/ggen /ggen && \
  apk del go git make gcc musl-dev linux-headers && \
  rm -rf /go-genchain && rm -rf /var/cache/apk/*

EXPOSE 17090
EXPOSE 60606

ENTRYPOINT ["/ggen"]
