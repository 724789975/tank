#! /bin/bash

ctr -n k8s.io image import match-server.tar.gz 

kubectl apply -f match-server.yaml 


