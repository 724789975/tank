#! /bin/bash

rm -rf auction-server.tar.gz

kubectl delete -f auction-server.yaml 

ctr -n k8s.io image rm docker.io/library/auction-server:v1.0.0