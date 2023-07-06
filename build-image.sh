docker buildx build -t 192.168.1.76:8099/evtrain/cvmart-daemon-arm64:v1.0 --platform linux/arm64 --build-arg  GNU_ARCH="aarch64" -f Dockerfile.filebeat . --load
docker build --build-arg TARGETARCH="amd64" --build-arg GNU_ARCH="x86_64" -t 192.168.1.76:8099/evtrain/cvmart-daemon-amd64:v1.0 -f Dockerfile.filebeat .

docker push 192.168.1.76:8099/evtrain/cvmart-daemon-amd64:v1.0
docker push 192.168.1.76:8099/evtrain/cvmart-daemon-arm64:v1.0
docker manifest rm 192.168.1.76:8099/evtrain/cvmart-daemon:v1.0
docker manifest create  192.168.1.76:8099/evtrain/cvmart-daemon:v1.0 \
192.168.1.76:8099/evtrain/cvmart-daemon-amd64:v1.0 \
192.168.1.76:8099/evtrain/cvmart-daemon-arm64:v1.0 -a --insecure

docker manifest annotate 192.168.1.76:8099/evtrain/cvmart-daemon:v1.0 \
192.168.1.76:8099/evtrain/cvmart-daemon-amd64:v1.0 --os linux --arch amd64

docker manifest annotate 192.168.1.76:8099/evtrain/cvmart-daemon:v1 \
192.168.1.76:8099/evtrain/cvmart-daemon-arm64:v1.0 --os linux --arch arm64

docker manifest push 192.168.1.76:8099/evtrain/cvmart-daemon:v1.0 --insecure