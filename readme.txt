
DFS is the simplest distributed file system written in golang.

[FlowChart]
1.Put|Post|Delete
Loadbalancer -> Data proxy 	-> Data backend servers 
							-> Metadata db

2.Get:
Loadbalancer -> Data proxy 	-> Data backend server 

3.Head:
Loadbalancer -> Data proxy 	-> Metadata db

[Require]
go 1.12 or above
elasticsearch 6.7 or above

[Run]
1. start Metadata db 
cd es
sh es.sh

2. src build
cd ../src
go mod init dfs
go build 
./dfs &

3.test
cd  ../test
sh test.sh

4.check
curl -XGET http://localhost:9200/backends/_search?pretty
curl -XGET http://localhost:9200/objects/_search?pretty


