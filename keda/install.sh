#!/bin/bash

helm repo add kedacore https://kedacore.github.io/charts
helm repo update
helm install keda kedacore/keda --namespace keda --create-namespace



# For uninstall
# kubectl delete $(kubectl get scaledobjects.keda.sh,scaledjobs.keda.sh -A \
#   -o jsonpath='{"-n "}{.items[*].metadata.namespace}{" "}{.items[*].kind}{"/"}{.items[*].metadata.name}{"\n"}')
# helm uninstall keda -n keda