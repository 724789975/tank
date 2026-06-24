#! /bin/bash

rm -rf item-manager.tar.gz

kubectl delete -f item-manager.yaml 

ctr -n k8s.io image rm docker.io/library/item-manager:v1.0.0