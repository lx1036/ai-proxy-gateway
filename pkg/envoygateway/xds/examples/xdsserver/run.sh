
# 1.
envoy -c envoy-xds.yaml --drain-time-s 1 -l debug

# 2.
go run . --debug=true --port=18000 --nodeID=test-id

# 3.
# 跳转到百度
curl localhost:10000


curl -s localhost:19000/config_dump | yq -P > envoy-config.yaml

