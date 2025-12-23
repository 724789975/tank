#! /bin/bash

ctr -n k8s.io image import user-server.tar.gz 

kubectl apply -f user-server.yaml 


