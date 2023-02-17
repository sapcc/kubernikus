kubectl -n kube-system delete svc kube-dns
kubectl -n kube-system delete sa kube-dns
kubectl -n kube-system delete cm kube-dns
kubectl -n kube-system delete deployment kube-dns
