#!/bin/sh

# 1. global
ESNAME=es
ESPATH=`pwd`

# 2. elasticsearch
echo "start elasticsearch..."
cd $ESPATH
grep "vm.max_map_count.*" /etc/sysctl.conf > /dev/null
if [ $? -eq 0 ]; then
        sed -i "s/vm.max_map_count.*/vm.max_map_count=262144/g" /etc/sysctl.conf 
else
        echo "" >>  /etc/sysctl.conf & echo "vm.max_map_count=262144" >> /etc/sysctl.conf
fi
docker pull elasticsearch:6.7.2
docker stop $ESNAME
docker rm -f $ESNAME
mkdir -p data
chmod 777 data -R
docker run -d --restart=always --name=$ESNAME --net=host --log-opt max-size=5m \
-e ES_JAVA_OPTS="-Xms1g -Xmx1g" -e "http.cors.enabled=true" -e "http.cors.allow-origin="*"" \
-v `pwd`/data:/usr/share/elasticsearch/data \
-v `pwd`/config/elasticsearch.yml:/usr/share/elasticsearch/config/elasticsearch.yml:ro \
-v /etc/localtime:/etc/localtime:ro \
elasticsearch:6.7.2
docker ps
