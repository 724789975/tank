#! /bin/bash

ctr -n k8s.io image import gate-server.tar.gz 

kubectl apply -f gate-server.yaml 


