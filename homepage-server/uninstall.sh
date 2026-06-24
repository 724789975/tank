#! /bin/bash

rm -rf homepage-server.tar.gz

kubectl delete -f homepage-server.yaml 

ctr -n k8s.io image rm docker.io/library/homepage-server:v1.0.0