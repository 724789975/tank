#! /bin/bash

rm -rf route-server.tar.gz

kubectl delete -f route-server.yaml 

ctr -n k8s.io image rm docker.io/library/route-server:v1.0.0
