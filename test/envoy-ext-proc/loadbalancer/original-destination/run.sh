







envoy -c ./envoy.yaml -l debug

go run .

curl localhost:10000/  -i
curl localhost:10000/unprotected -i

cd /Users/lx1036/Code/k8s/envoyproxy-gateway/tmp/envoy/grpc-ext-proc
python3 -m http.server 8899

curl localhost:10000/ -i -v -H "target-pod: 127.0.0.1:8899"


