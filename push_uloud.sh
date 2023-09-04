VERSION=v1.1
docker tag 192.168.1.76:8099/evtrain/cvmart-daemon-amd64:${VERSION} uhub.service.ucloud.cn/evtrain/cvmart-daemon-amd64:${VERSION}
docker tag 192.168.1.76:8099/evtrain/cvmart-daemon-arm64:${VERSION} uhub.service.ucloud.cn/evtrain/cvmart-daemon-arm64:${VERSION}
docker push uhub.service.ucloud.cn/evtrain/cvmart-daemon-amd64:${VERSION}
docker push uhub.service.ucloud.cn/evtrain/cvmart-daemon-arm64:${VERSION}