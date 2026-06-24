#! /bin/bash

ctr -n k8s.io image import auction-server.tar.gz 

kubectl apply -f auction-server.yaml 