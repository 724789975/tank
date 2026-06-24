#! /bin/bash

ctr -n k8s.io image import item-manager.tar.gz 

kubectl apply -f item-manager.yaml 