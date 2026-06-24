#! /bin/bash

ctr -n k8s.io image import homepage-server.tar.gz 

kubectl apply -f homepage-server.yaml 