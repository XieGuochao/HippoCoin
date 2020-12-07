FROM golang:1.15

WORKDIR /go/src/HippoCoin
COPY . .

# install gettext for envsubst
RUN sed -i s@/deb.debian.org/@/mirrors.cloud.tencent.com/@g /etc/apt/sources.list
RUN apt-get clean
RUN apt-get update
RUN apt-get install -y gettext-base

ENV GOPROXY https://mirrors.cloud.tencent.com/go/
RUN go get -d -v ./...
RUN go install -v ./...
RUN go build -o coin *.go

ENV curve=P224
ENV miningthreads=1
ENV broadcastqueuelen 10
ENV miningcapacity 10
ENV mininginterval 15
ENV miningttl 7200
ENV protocol tcp
ENV maxneighbors 5
ENV updatetimebase 10
ENV updatetimerand 10

ENV registeraddress localhost:9325
ENV registerprotocol tcp

ENV localmode false

ENV debugfiletemplate ./log/host1-debug-%s.log
ENV infofiletemplate STDOUT

ENV uiport 8080
ENV listenerport 9000

EXPOSE 8080
EXPOSE 9000

CMD envsubst < host-template.yml > host.yml && ./coin