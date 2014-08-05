FROM dev-docker.helloreverb.com/base:latest
MAINTAINER Reverb Technologies, Inc 

ADD ./agora.linux /agora
ADD ./static /static
ADD ./libmesos-0.19.0-rvb3.so /lib/libmesos-0.19.0-rvb3.so
ADD ./libmesos.la /lib/libmesos.la

EXPOSE 8000

VOLUME ["/etc/agora"]
VOLUME ["/data"]
VOLUME ["/logs"]
VOLUME ["/work"]

CMD ["/agora --conf=/etc/agora"]
