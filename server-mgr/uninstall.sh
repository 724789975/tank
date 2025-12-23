#! /bin/bash

rm -rf server-mgr.tar.gz

kubectl delete -f server-mgr.yaml 

ctr -n k8s.io image rm docker.io/library/server-mgr:v1.0.0

