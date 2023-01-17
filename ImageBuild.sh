docker buildx build -t 192.168.1.186:8099/evtrain/cvmart-log:v1 --platform linux/arm64 -f Dockerfile.filebeat . --load
docker  build --build-arg TARGETARCH="amd64" -t 192.168.1.186:8099/evtrain/cvmart-log-amd64:v1 -f Dockerfile.filebeat .

docker manifest create  192.168.1.186:8099/evtrain/cvmart-log:v1 \
192.168.1.186:8099/evtrain/cvmart-log-amd64:v1 \
192.168.1.186:8099/evtrain/cvmart-log-arm64:v1 -a


docker manifest annotate 192.168.1.186:8099/evtrain/cvmart-log:v1 \
192.168.1.186:8099/evtrain/cvmart-log-amd64:v1 --os linux --arch amd64

docker manifest annotate 192.168.1.186:8099/evtrain/cvmart-log:v1 \
192.168.1.186:8099/evtrain/cvmart-log-arm64:v1 --os linux --arch arm64