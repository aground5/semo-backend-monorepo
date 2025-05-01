helm install semo-authn ./semo-authn-server -n your-namespace --create-namespace
helm upgrade --install semo-authn ./helm -n semo --create-namespace -f values.yaml