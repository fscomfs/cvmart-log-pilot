VERSION=v1.2
REGISTRY=192.168.1.76:8099/evtrain
IMAGE_NAME=cvmart-daemon
docker buildx build -t ${REGISTRY}/cvmart-daemon-arm64:${VERSION} --platform linux/arm64 --build-arg  GNU_ARCH="aarch64" -f Dockerfile.filebeat . --load
docker build --build-arg TARGETARCH="amd64" --build-arg GNU_ARCH="x86_64" -t ${REGISTRY}/cvmart-daemon-amd64:${VERSION} -f Dockerfile.filebeat .

docker push ${REGISTRY}/${IMAGE_NAME}-amd64:${VERSION}
docker push ${REGISTRY}/${IMAGE_NAME}-arm64:${VERSION}
docker manifest rm ${REGISTRY}/${IMAGE_NAME}:${VERSION}
docker manifest create  ${REGISTRY}/${IMAGE_NAME}:${VERSION} \
${REGISTRY}/${IMAGE_NAME}-amd64:${VERSION} \
${REGISTRY}/${IMAGE_NAME}-arm64:${VERSION} -a --insecure

docker manifest annotate ${REGISTRY}/${IMAGE_NAME}:${VERSION} \
${REGISTRY}/${IMAGE_NAME}-amd64:${VERSION} --os linux --arch amd64

docker manifest annotate ${REGISTRY}/${IMAGE_NAME}:${VERSION} \
${REGISTRY}/${IMAGE_NAME}-arm64:${VERSION} --os linux --arch arm64

docker manifest push ${REGISTRY}/${IMAGE_NAME}:${VERSION} --insecure