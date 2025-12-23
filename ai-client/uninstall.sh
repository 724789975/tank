#! /bin/bash

rm -rf ai-client.tar.gz

#kubectl delete -f ai-client.yaml 

ctr -n k8s.io image rm docker.io/library/ai-client:v1.0.0

