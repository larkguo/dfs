#!/bin/sh

# 1. global
TESTPATH=`pwd`

# 2. set server backends
curl -XDELETE localhost:9200/backends/

curl -XPUT http://127.0.0.1:9200/backends/_doc/http_localhost:2020 -H 'Content-Type: application/json' -d '{
	"backend": "http://localhost:2020",
	"alive" : 1,
	"policy": 0,
	"magic": 0.0,
	"size": 0,
	"proxy": 0,
	"weight": 1
}'

curl -XPUT http://127.0.0.1:9200/backends/_doc/http_127.0.0.1:2021 -H 'Content-Type: application/json' -d '{
	"backend": "http://127.0.0.1:2021",
	"alive" : 1,
	"policy": 0,
	"magic": 0.0,
	"size": 0,
	"proxy": 0,
	"weight": 1
}'


# 3. test
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

# 4. check
curl http://localhost:9200/backends/_search?pretty
curl http://localhost:9200/objects/_search?pretty
