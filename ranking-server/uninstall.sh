#! /bin/bash

rm -rf ranking-server.tar.gz

kubectl delete -f ranking-server.yaml 

ctr -n k8s.io image rm docker.io/library/ranking-server:v1.0.0
