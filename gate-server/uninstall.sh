#! /bin/bash

rm -rf gate-server.tar.gz

kubectl delete -f gate-server.yaml 

ctr -n k8s.io image rm docker.io/library/gate-server:v1.0.0

