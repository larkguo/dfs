
DFS is the simplest distributed file system written in golang.

[Require]
go 1.12 or above
elasticsearch 6.7 or above

[Elasticsearch]
cd es
docker pull elasticsearch:6.7.2
sh es.sh

[Src Build]
cd src
sh src.sh

[Test]
cd test
sh test.sh

[Check]
curl http://localhost:9200/backends/_search?pretty
curl http://localhost:9200/objects/_search?pretty


