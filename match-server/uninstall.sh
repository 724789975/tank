#! /bin/bash

rm -rf match-server.tar.gz

kubectl delete -f match-server.yaml 

ctr -n k8s.io image rm docker.io/library/match-server:v1.0.0

