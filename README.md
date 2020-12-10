# HippoCoin

![Hippo](log/Hippo.jpg)

A blockchain implementation for CIE6125, Fall 2020, CUHK(SZ).

# Requirements

1. Go 1.15 is required.
2. Docker is preferred.
3. Make sure you have set up proxies properly if you are accessing from China.

# Build

## Build HippoRegister

1. `git clone https://github.com/XieGuochao/HippoCoinRegister.git`
2. `cd HippoCoinRegister`
3. `go install`
4. `make server`
5. Make sure to run it __BEFORE__ running the host!

Note: HippoRegister is running on port 9325 by default. The reason I choose this port is because my girlfriend's birthday is on March 25 :)

## Build HippoCoin

1. `git clone https://github.com/XieGuochao/HippoCoin.git`
2. `cd HippoCoin`
3. `go install`
4. `go build -o coin`
5. Now it has been compiled into `./coin`.
6. Change the settings in `host.yml` and run `./coin` (by default it uses `host.yml`) or `./coin YOURYML.yml`.
7. Make sure to run your register __BEFORE__ running the host!
8. Now the web client is running on your `ui-port` (8080 by default).

# Run

## Build from source

You may refer to the [_Build Part_](#build).

## Run from Docker

1. Register: `sudo docker run -p 9325:9325 -d ccr.ccs.tencentyun.com/hippocoin/register`
2. Host: `sudo docker run -p 10001:8080 -p 11001:11001 --env registeraddress=172.17.0.2:9325 \
    --env infofiletemplate=./log/host$i-info-%s.log \
    --env debugfiletemplate=./log/host$i-debug-%s.log \
    --env listenerport=11001 \
    --expose 11001 \
    --cpus=2 \
    -d \
    ccr.ccs.tencentyun.com/hippocoin/coin`
    
    You may need to change `registeraddress` if you have modified it. And your web client UI will be on port `10001`.

## Run from Bash Scripts (Docker Required)

The following scripts are for you to conveniently run multiple Docker containers:

1. `create-register.sh`
2. `create-hosts.sh i j`: Create ordinary hosts from index `i` to index `j` and the client ports are from `10000+i` to `10000+j`.
3. `create-large.sh i j t`: Create large hosts from index `i` to index `j` with `t` mining threads. Client ports are from `10000+i` to `10000+j`.
4. `delete-all-hosts.sh`: Stop and delete all hosts containers.
5. `delete-register.sh`: Stop and delete register container.

__Warning__: make sure you do not have hosts' ports overlap.

# Image

If you want an image of Ubuntu 18.04 installed with all requirements and ready to run HippoCoin, please email me: [guochaoxie@link.cuhk.edu.cn](mailto:guochaoxie@link.cuhk.edu.cn).

# Acknowledgement

This project is supported by [Apartsa Co. Ltd.](https://apartsa.com/).

# Potential Bugs

The following are potential bugs. Welcome to contribute and fix them.

1. P2P network may fail to find neighbors.