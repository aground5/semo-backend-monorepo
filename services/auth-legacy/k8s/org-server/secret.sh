kubectl create secret generic oxhr-auth-server-config-with-secrets \
  --namespace oxhr \
  --from-file=configs.yaml=configs/file/configs.yaml