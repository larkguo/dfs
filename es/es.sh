#!/bin/sh

grep "vm.max_map_count.*" /etc/sysctl.conf > /dev/null
if [ $? -eq 0 ]; then
        sed -i "s/vm.max_map_count.*/vm.max_map_count=262144/g" /etc/sysctl.conf 
else
        echo "" >>  /etc/sysctl.conf & echo "vm.max_map_count=262144" >> /etc/sysctl.conf
fi


docker load -i elasticsearch.tar
docker stop es
docker rm -f es
mkdir -p data
chmod 777 data -R
docker run -d --restart=always --name=es --net=host --log-opt max-size=5m \
-e ES_JAVA_OPTS="-Xms1g -Xmx1g" -e "http.cors.enabled=true" -e "http.cors.allow-origin="*"" \
-v `pwd`/data:/usr/share/elasticsearch/data \
-v `pwd`/config/elasticsearch.yml:/usr/share/elasticsearch/config/elasticsearch.yml:ro \
-v /etc/localtime:/etc/localtime:ro \
elasticsearch
