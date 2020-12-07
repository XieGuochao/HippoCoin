#!/bin/bash

for i in $(seq 1 $1)
do 
    echo "apiVersion: v1
kind: Service
metadata:
  annotations:
    service.kubernetes.io/service.extensiveParameters: '{\"AddressIPVersion\":\"IPV4\"}'
  managedFields:
  - apiVersion: v1
    manager: tke-apiserver
    operation: Update
  - apiVersion: v1
    manager: service-controller
    operation: Update
  name: h$i
  namespace: default
  selfLink: /api/v1/namespaces/default/services/h$i
spec:
  externalTrafficPolicy: Cluster
  ports:
  - name: 8080-8080-tcp
    port: 8080
    protocol: TCP
    targetPort: 8080
  - name: 9000-9000-tcp
    port: 9000
    protocol: TCP
    targetPort: 9000
  selector:
    k8s-app: h$i
    qcloud-app: h$i
  sessionAffinity: None
  type: LoadBalancer" | kubectl create --validate=false -f -
done