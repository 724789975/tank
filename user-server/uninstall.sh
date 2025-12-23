#! /bin/bash

rm -rf user-server.tar.gz

kubectl delete -f user-server.yaml 

ctr -n k8s.io image rm docker.io/library/user-server:v1.0.0

