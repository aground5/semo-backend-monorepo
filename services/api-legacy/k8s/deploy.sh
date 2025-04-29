helm upgrade --install semo-backend ./helm -n semo-staging -f ./staging-values.yaml --create-namespace

helm upgrade --install semo-backend ./helm -n semo -f ./values.yaml --create-namespace
