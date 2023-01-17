cvmart_logs_custom_config=trackNo=uuid;minioObjName=uuid/name.log
cvmart_logs_minioObjName=/uuid/name.log
cvmart_logs_trackNo=uuid
cvmart_logs_limitSize=50000

cvmart_log_skip


DOCKER_HOST
DOCKER_API_VERSION




cvmart_log_skip


docker run -d --env cvmart_logs_custom_config="cvmart.fields.trackNo=uuid2;cvmart.fields.limitSize=50000;cvmart.fields.minioObjName=/uuid2/name2.log" --env PILOT_LOG_PREFIX=cvmart --env cvmart_logs_cvmart=stdout -v /cvmart-log/:/log a8780b506fa4  bash -c "chmod  +x /log/log.sh && bash /log/log.sh"

docker run -d --env cvmart_logs_custom_config="cvmart.fields.trackNo=uuid3;cvmart.fields.limitSize=50000;cvmart.fields.minioObjName=/uuid3/name2.log" --env PILOT_LOG_PREFIX=cvmart --env cvmart_logs_cvmart=stdout -v /cvmart-log/:/log a8780b506fa4  bash -c "chmod  +x /log/log.sh && bash /log/log.sh"


-c /cvmart-log/beats/filebeat/filebeat.yml


/etc/filebeat/filebeat.yml



/data/filebeat/filebeat/log.json


BUILD_DIR?=$(shell pwd)/build
gox -osarch "linux/amd64 linux/arm64 linux/arm" -output={{.Dir}}_{{.OS}}_{{.Arch}}

export PILOT_TYPE=filebeat
export PILOT_LOG_PREFIX=cvmart
export JWT_SEC="111"
export KUBERNETES_SERVICE_HOST=192.168.1.131
export KUBERNETES_SERVICE_PORT=6443
export KUBERNETES_TOKEN=
export MINIO_URL="192.168.1.175:9000"
export MINIO_USERNAME=admin
export MINIO_PASSWORD="admin123"
export BUCKET="cvmart-log"
log-pilot --template="/config/filebeat/filebeat.tpl" --base="/"


docker  build --build-arg ARCH="amd64" -t 192.168.1.186:8099/evtrain/cvmart-log:v1 -f Dockerfile.filebeat .

docker buildx build -t 192.168.1.186:8099/evtrain/cvmart-log:v1 --platform linux/amd64,linux/arm64 -f Dockerfile.filebeat .


docker run -it --rm -d \
--env PILOT_TYPE=filebeat \
--env PILOT_LOG_PREFIX=cvmart \
--env JWT_SEC="111" \
--env KUBERNETES_SERVICE_HOST=192.168.1.131 \
--env KUBERNETES_SERVICE_PORT=6443 \
--env KUBERNETES_TOKEN="eyJhbGciOiJSUzI1NiIsImtpZCI6Ii1EQkxmRVlFMnFBNm1xcHk3U2NhSm0xUGhaZnlsT2dZeFNIZUFRZzBLU0UifQ.eyJpc3MiOiJrdWJlcm5ldGVzL3NlcnZpY2VhY2NvdW50Iiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9uYW1lc3BhY2UiOiJrdWJlLXN5c3RlbSIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VjcmV0Lm5hbWUiOiJhZG1pbi10b2tlbi14bjh0cCIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VydmljZS1hY2NvdW50Lm5hbWUiOiJhZG1pbiIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VydmljZS1hY2NvdW50LnVpZCI6IjYyOTQ0YmQwLTNhN2MtNDNlYy05MTYyLTc3ZGNjYTEwMDU3NyIsInN1YiI6InN5c3RlbTpzZXJ2aWNlYWNjb3VudDprdWJlLXN5c3RlbTphZG1pbiJ9.JCyjrKbVMVis28DIAjp1L9BwlqT3XXGrTHH_oUN_4Xu6gcOP2GOokg9S66CXZR7CSPTtTWbpRFHu3KyoISQFl5TxatDGrHvEjbMtcugwHBTW6yrfxJs_woN4QphlFq5wBzmwcpvC1MXuj3VTIRvabnivfL3wa2qw3iccP8eYSPpaySVKChu60WW_oYMrvVOL3PG01DlWY2PuVS6-uHliCal5_lY22VWKo8AROpoe8tWVa5YEeY45LEe9bsK-WXqY9OweN3PLOGELpjAeY5wc5GJCsACm9Jvv43CfuGopz7rKD005dlfojY4GvF9IVnEMSXFtv0ZtDRSMwPqECubsnQ" \
--env MINIO_URL="192.168.1.175:9000" \
--env MINIO_USERNAME="admin" \
--env MINIO_PASSWORD="admin123" \
--env BUCKET="cvmart-log" \
-v /var/run/docker.sock:/var/run/docker.sock \
-v /dev-docker:/dev-docker \
-v /log-monitor/out:/out \
-v /log-monitor/config:/config \
-p 888:888 \
--name cvmart-log \
192.168.1.186:8099/evtrain/cvmart-log:v1




docker images|grep none|awk '{print $3}'|xargs -I {} docker rmi {} -f



docker manifest create  192.168.1.186:8099/evtrain/cvmart-log:v1 \
192.168.1.186:8099/evtrain/cvmart-log-amd64:v1 \
192.168.1.186:8099/evtrain/cvmart-log-arm64:v1


docker manifest annotate 192.168.1.186:8099/evtrain/cvmart-log:v1 \
192.168.1.186:8099/evtrain/cvmart-log-amd64:v1 --os linux --arch amd64

docker manifest annotate 192.168.1.186:8099/evtrain/cvmart-log:v1 \
192.168.1.186:8099/evtrain/cvmart-log-arm:v1 --os linux --arch arm64




