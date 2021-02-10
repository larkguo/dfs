
DFS is the simplest distributed file system written in golang.

	v1.0 Loadbalancing by capacity
		By default, the server with the smallest StoredSize/Weight is selected for forwarding

	v2.0 ErasureCode 

[FlowChart]
	1.Put|Post|Delete
	Client -> Loadbalancer -> BackendServer& MetadataDB

	2.Get:
	Client -> Loadbalancer -> BackendServer 

	3.Head:
	Client -> Loadbalancer -> MetadataDB

[Require]
	go 1.12 or above
	elasticsearch 6.7 or above

[Run]
	1. start MetadataDB 
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


