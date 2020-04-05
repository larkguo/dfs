#!/bin/sh

# 1. global
DFSNAME=dfs
ESNAME=es
ROOTPATH=`pwd`
ESPATH=$ROOTPATH/es
SRCPATH=$ROOTPATH/src

# 2. elasticsearch
echo "start elasticsearch..."
cd $ESPATH
grep "vm.max_map_count.*" /etc/sysctl.conf > /dev/null
if [ $? -eq 0 ]; then
        sed -i "s/vm.max_map_count.*/vm.max_map_count=262144/g" /etc/sysctl.conf 
else
        echo "" >>  /etc/sysctl.conf & echo "vm.max_map_count=262144" >> /etc/sysctl.conf
fi
docker load -i elasticsearch.tar
docker stop $ESNAME
docker rm -f $ESNAME
mkdir -p data
chmod 777 data -R
docker run -d --restart=always --name=$ESNAME --net=host --log-opt max-size=5m \
-e ES_JAVA_OPTS="-Xms1g -Xmx1g" -e "http.cors.enabled=true" -e "http.cors.allow-origin="*"" \
-v `pwd`/data:/usr/share/elasticsearch/data \
-v `pwd`/config/elasticsearch.yml:/usr/share/elasticsearch/config/elasticsearch.yml:ro \
-v /etc/localtime:/etc/localtime:ro \
elasticsearch
docker ps
sleep 30

# 3. dfs source
echo "start dfs..."
cd $SRCPATH

pkill -9 $DFSNAME
go mod init $DFSNAME
go build 
chmod +x $DFSNAME
./$DFSNAME -l :80  &

# 4. set server backends
curl -XDELETE localhost:9200/backends/

curl -XPUT http://127.0.0.1:9200/backends/_doc/http_localhost:2020 -H 'Content-Type: application/json' -d '{
	"@timestamp" : "2008-02-22T11:06:00.000Z",
	"backend": "http://localhost:2020",
	"alive" : 1,
	"policy": 0,
	"magic": 0.0,
	"size": 0,
	"proxy": 0,
	"weight": 1
}'

curl -XPUT http://127.0.0.1:9200/backends/_doc/http_127.0.0.1:2021 -H 'Content-Type: application/json' -d '{
	"@timestamp" : "2008-02-22T11:06:00.000Z",
	"backend": "http://127.0.0.1:2021",
	"alive" : 1,
	"policy": 0,
	"magic": 0.0,
	"size": 0,
	"proxy": 0,
	"weight": 1
}'

# curl http://localhost:9200/backends/_search?pretty


# 4. test
curl -XDELETE localhost:9200/objects/

sleep 1
OBJECT1="/objects/test1"
OBJECT1_SHA256=`echo  -n $OBJECT1 | sha256sum -t | cut -d ' ' -f1`
curl -XPUT -v http://localhost$OBJECT1  -d "$OBJECT1"  -H "Digest: sha-256=$OBJECT1_SHA256"
sleep 1
curl -XGET  -v http://localhost$OBJECT1 
# curl -XDELETE -v http://localhost$OBJECT1

OBJECT2="/objects/test2"
OBJECT2_SHA256=`echo  -n $OBJECT2 | sha256sum -t | cut -d ' ' -f1`
curl -XPUT -v http://localhost$OBJECT2  -d "$OBJECT1" -H "Digest: sha-256=$OBJECT1_SHA256"
sleep 1
curl -XGET  -v http://localhost$OBJECT2  
# curl -XDELETE -v http://localhost$OBJECT2
# curl http://localhost:9200/objects/_search?pretty
