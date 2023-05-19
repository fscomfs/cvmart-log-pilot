##

CGO_ENABLED=1 make

docker run -d --env cvmart_logs_stdout=stdout --env cvmart_logs_custom_config='stdout.fields.trackNo=2;stdout.fields.limitSize=300000;stdout.fields.minioObjName=/2/2.log'  cd15c3be83e4 