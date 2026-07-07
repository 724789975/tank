#! /bin/bash

ctr -n k8s.io image import ranking-server.tar.gz 

kubectl apply -f ranking-server.yaml 
