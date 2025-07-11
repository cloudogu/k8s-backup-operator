#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

docker run -d --name minio -p 9000:9000 -p 9090:9090 -e "MINIO_ROOT_USER=admin123" -e "MINIO_ROOT_PASSWORD=admin123" quay.io/minio/minio server /data --console-address ":9090"


docker exec -t minio mc alias set minio http://localhost:9000 admin123 admin123
docker exec -t minio mc admin accesskey create minio/ --access-key MY-ACCESS-KEY --secret-key MY-ACCESS-SECRET123
docker exec -t minio mc admin accesskey create minio/ --access-key MY-VELERO-ACCESS-KEY --secret-key MY-VELERO.ACCESS-SECRET123
docker exec -t minio mc mb minio/velero --region minio-default
docker exec -t minio mc mb minio/longhorn --region minio-default