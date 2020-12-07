#!/bin/bash

for i in $(seq $1 $2)
do
    sudo docker run -p $(($i + 10000)):8080 -p $(($i + 11000)):$(($i + 11000)) --env registeraddress=172.17.0.2:9325 \
    --env infofiletemplate=./log/host$i-info-%s.log \
    --env debugfiletemplate=./log/host$i-debug-%s.log \
    --env listenerport=$(($i + 11000)) \
    --expose $(($i + 11000)) \
    --cpus=2 \
    -d \
    hippocoin.tencentcloudcr.com/hippo/coin
done