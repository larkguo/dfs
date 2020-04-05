
dfs is the simplest distributed file system written in golang.


1. go mod build & test
sh dfs.sh

2. metadata(backend servers & files)
curl http://localhost:9200/backends/_search?pretty
curl http://localhost:9200/objects/_search?pretty


