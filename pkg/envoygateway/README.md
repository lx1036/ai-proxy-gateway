



1. k8s provider watch gateway-api resources -> publish GatewayAPIResources message

                                                                        -> publish InfraIR message  ---> infra subscribe InfraIR 生成 Envoy Pod 相关资源
                                                                     /  
                                                                    /
2. -> gateway-api translator subscribe GatewayAPIResources message -> 
                                                                    \
                                                                     \
                                                                        -> publish XdsIR message   ---> xds subscribe XdsIR 生成 xDS Server 资源，来动态更新 Envoy Pod 配置 




