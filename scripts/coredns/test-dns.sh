kubectl -n default apply -f dnsutils.yaml
sleep 30
kubectl -n default exec -i -t dnsutils -- nslookup kubernetes.default

