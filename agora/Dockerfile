FROM busybox:ubuntu-14.04
MAINTAINER Reverb Technologies, Inc 

ADD ./dist/agora.linux /agora
ADD ./dist/static /static

EXPOSE 8000

VOLUME ["/etc/agora"]

CMD ["/agora"]
