docker tag 192.168.1.76:8099/evtrain/cvmart-daemon-amd64:v1.0 uhub.service.ucloud.cn/evtrain/cvmart-daemon-amd64:v1.0
docker tag 192.168.1.76:8099/evtrain/cvmart-daemon-arm64:v1.0 uhub.service.ucloud.cn/evtrain/cvmart-daemon-arm64:v1.0
docker push uhub.service.ucloud.cn/evtrain/cvmart-daemon-amd64:v1.0
docker push uhub.service.ucloud.cn/evtrain/cvmart-daemon-arm64:v1.0