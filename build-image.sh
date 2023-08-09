VERSION=v1.1
docker buildx build -t 192.168.1.76:8099/evtrain/cvmart-daemon-arm64:${VERSION} --platform linux/arm64 --build-arg  GNU_ARCH="aarch64" -f Dockerfile.filebeat . --load
docker build --build-arg TARGETARCH="amd64" --build-arg GNU_ARCH="x86_64" -t 192.168.1.76:8099/evtrain/cvmart-daemon-amd64:${VERSION} -f Dockerfile.filebeat .

docker push 192.168.1.76:8099/evtrain/cvmart-daemon-amd64:${VERSION}
docker push 192.168.1.76:8099/evtrain/cvmart-daemon-arm64:${VERSION}
docker manifest rm 192.168.1.76:8099/evtrain/cvmart-daemon:${VERSION}
docker manifest create  192.168.1.76:8099/evtrain/cvmart-daemon:${VERSION} \
192.168.1.76:8099/evtrain/cvmart-daemon-amd64:${VERSION} \
192.168.1.76:8099/evtrain/cvmart-daemon-arm64:${VERSION} -a --insecure

docker manifest annotate 192.168.1.76:8099/evtrain/cvmart-daemon:${VERSION} \
192.168.1.76:8099/evtrain/cvmart-daemon-amd64:${VERSION} --os linux --arch amd64

docker manifest annotate 192.168.1.76:8099/evtrain/cvmart-daemon:${VERSION} \
192.168.1.76:8099/evtrain/cvmart-daemon-arm64:${VERSION} --os linux --arch arm64

docker manifest push 192.168.1.76:8099/evtrain/cvmart-daemon:${VERSION} --insecure