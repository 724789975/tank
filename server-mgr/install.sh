#! /bin/bash

ctr -n k8s.io image import server-mgr.tar.gz 

kubectl apply -f server-mgr.yaml 


