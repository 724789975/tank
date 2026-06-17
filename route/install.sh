#! /bin/bash

ctr -n k8s.io image import route-server.tar.gz 

kubectl apply -f route-server.yaml 
