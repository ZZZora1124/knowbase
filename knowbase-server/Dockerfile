FROM iregistry.baidu-int.com/baidu-base/centos:7

MAINTAINER hekai09@baidu.com

LABEL IMAGE_NAME=comate-stack/knowbase-server

COPY output/knowbase-server /app/knowbase-server

RUN mkdir /logs
RUN chown work:work /app /logs

ENTRYPOINT ["/app/knowbase-server"]
