FROM alpine:latest
ARG config
COPY ./logs /data/msgPushSite/logs
COPY ./config /data/msgPushSite/config
COPY ./msgPushSite /data/msgPushSite/msgpushsite
WORKDIR /data/msgPushSite
VOLUME ["/data/msgPushSite/logs"]

RUN apk update \
    && apk upgrade \
    && apk add --no-cache  \
    && apk add ca-certificates \
    && apk add update-ca-certificates 2>/dev/null || true \
    && apk add tzdata \
    && ln -fs /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && mv ${config} ./config/active.ini

CMD ["/data/msgPushSite/msgpushsite", "--config=./config/active.ini"]