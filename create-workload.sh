#!/bin/bash

for i in $(seq 1 $1)
do
echo "apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: \"1\"
  generation: 1
  labels:
    k8s-app: h$i
    qcloud-app: h$i
  managedFields:
  - apiVersion: apps/v1
    manager: tke-apiserver
    operation: Update
  - apiVersion: apps/v1
    manager: kube-controller-manager
    operation: Update
  name: h$i
  namespace: default
spec:
  replicas: 1
  revisionHistoryLimit: 10
  selector:
    matchLabels:
      k8s-app: h$i
      qcloud-app: h$i
  strategy:
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
    type: RollingUpdate
  template:
    metadata:
      creationTimestamp: null
      labels:
        k8s-app: h$i
        qcloud-app: h$i
    spec:
      containers:
      - env:
        - name: hipporegister
          value: $2
        - name: infofiletemplate
          value: ./log/host-info-%s.log
        image: hippocoin.tencentcloudcr.com/hippo/coin
        imagePullPolicy: Always
        name: h$i
        resources:
          limits:
            cpu: 3
            memory: 1Gi
          requests:
            cpu: 3
            memory: 1Gi
        securityContext:
          privileged: false
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 30
" | kubectl create --validate=false -f -
done