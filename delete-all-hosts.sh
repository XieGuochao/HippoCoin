#!/bin/bash

sudo docker rm $(sudo docker stop $(sudo docker ps -a -q --filter ancestor=hippocoin.tencentcloudcr.com/hippo/coin --format="{{.ID}}"))