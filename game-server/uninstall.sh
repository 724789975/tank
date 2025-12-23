#! /bin/bash

rm -rf game-server.tar.gz

ctr -n k8s.io image rm docker.io/library/game-server:v1.0.0

