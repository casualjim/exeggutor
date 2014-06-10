CC=gcc-4.8
CXX=g++-4.8

all: dist

build-binary: 
	CC=gcc-4.8 CXX=g++-4.8 go build

build-frontend:
	cd static \
	grunt build  \
	cd .. \

dist: build-binary build-frontend
	mkdir -p dist
	mv agora dist
	mv static/build dist/static
	mv dist agora
	tar -czf agora.tar.gz agora
	rm -rf agora

upload: 
	scp agora.tar.gz dev-mesos.helloreverb.com:~

clean:
	rm agora
	rm static/build

distclean: clean
	rm -rf dist	
