package db

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type DbClient struct { // inherit  Handler.ServeHTTP
}

var g_HttpClient *http.Client = createHTTPClient()
var g_DbURL string = "http://192.168.73.11:9200"

type ObjectItem struct {
	Timestamp string `json:"@timestamp"`
	Name      string `json:"name"`
	Backend   string `json:"backend"`
	Size      uint64 `json:"size"`
	Hash      string `json:"hash"`
}
type ObjectHit struct {
	Item ObjectItem `json:"_source"`
}
type ObjectResult struct {
	ScrollID string `json:"_scroll_id"`
	Hits     struct {
		Total int         `json:"total"`
		Hits  []ObjectHit `json:"hits"`
	}
}

type BackendItem struct {
	Timestamp string  `json:"@timestamp"`
	Backend   string  `json:"backend"`
	Alive     uint    `json:"alive"`
	Policy    int     `json:"policy"`
	Magic     float32 `json:"magic"`
	Size      uint64  `json:"size"`
	Proxy     uint64  `json:"proxy"`
	Weight    uint8   `json:"weight"`
}
type BackendHit struct {
	Item BackendItem `json:"_source"`
}
type BackendResult struct {
	ScrollID string `json:"_scroll_id"`
	Hits     struct {
		Total int          `json:"total"`
		Hits  []BackendHit `json:"hits"`
	}
}

// createHTTPClient for connection re-use
func createHTTPClient() *http.Client {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   2,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	return client
}

func (b *DbClient) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodPut, http.MethodPost:
		dbPutObject(w, r)
	case http.MethodDelete:
		dbDeleteObject(w, r)
	case http.MethodHead:
		DbGetObjectInfo(w, r)
	}
}
func hashFromHeaderGet(h http.Header) string {
	digest := h.Get("Digest")
	if len(digest) < 9 {
		digest = h.Get("digest")
		if len(digest) < 9 {
			return ""
		}
	}
	if strings.ToLower(digest[:8]) != "sha-256=" {
		return ""
	}
	return digest[8:]
}

// ================================= object =================================
/*
curl -XPUT -v http://192.168.73.1/objects/test1  -d"bucket1/object1" -H 'Digest: sha-256=35bfdc4784a46633f931e31644383d85ba88b43d25113cc262220bb727d89be1'
*/
func dbPutObject(w http.ResponseWriter, r *http.Request) {

	url := g_DbURL + "/objects/_doc"
	backend := r.URL.Scheme + "://" + r.URL.Host
	name := r.URL.Path
	timestr := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	size, e := strconv.ParseUint(r.Header.Get("content-length"), 0, 64)
	hash := hashFromHeaderGet(r.Header)
	doc := fmt.Sprintf(`{"@timestamp":"%s","name":"%s","backend":"%s","size":%d,"hash":"%s"}`,
		timestr, name, backend, size, hash)
	request, e := http.NewRequest("POST", url, strings.NewReader(doc))
	if e != nil {
		fmt.Println(e.Error())
		http.Error(w, e.Error(), http.StatusServiceUnavailable)
		return
	}
	request.Header.Set("Content-Type", "application/json")

	resp, e := g_HttpClient.Do(request)
	if e != nil {
		fmt.Println("write to es:", e.Error())
		http.Error(w, e.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	io.Copy(ioutil.Discard, resp.Body)

	// Statistics
	UpdateBackendStatistics(r.URL.Scheme, r.URL.Host, size, 1)
}

/*
curl -X POST "localhost:9200/objects/_doc/_delete_by_query" -H 'Content-Type: application/json' -d'{
  "query": {"match_phrase": {"name": "/objects/test1"}}
}'
*/
func dbDeleteObject(w http.ResponseWriter, r *http.Request) {

	url := g_DbURL + "/objects/_delete_by_query"
	name := r.URL.Path
	doc := fmt.Sprintf(`{ "query": {"match_phrase": {"name": "%s"}} }`, name)
	request, e := http.NewRequest("POST", url, strings.NewReader(doc))
	if e != nil {
		fmt.Println("db ", e.Error())
		http.Error(w, e.Error(), http.StatusServiceUnavailable)
		return
	}
	fmt.Println(url, doc)
	request.Header.Set("Content-Type", "application/json")
	resp, e := g_HttpClient.Do(request)
	if e != nil {
		fmt.Println("write to es:", e.Error())
		http.Error(w, e.Error(), http.StatusServiceUnavailable)
	}
	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

/*
curl -XGET http://localhost:9200/objects/_search?pretty -H 'Content-Type: application/json' -d '{
	 "size": 1,"query": {"bool": {"must": [ {"match_phrase": {"name": "/objects/test1"}}]}}
}'
*/
func DbGetObjectInfo(w http.ResponseWriter, r *http.Request) {

	var result ObjectResult
	var rsp ObjectItem

	defer func() {
		buf, e := json.Marshal(&rsp)
		if e != nil {
			fmt.Println("Marshal error:", e.Error())
			w.WriteHeader(http.StatusInternalServerError)
		}
		fmt.Println("info:", rsp)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(buf))
	}()

	url := fmt.Sprintf("%s", g_DbURL+"/objects/_search")
	name := r.URL.Path
	doc := fmt.Sprintf(`{ "size": 1,"query": {"bool": {"must": [ {"match_phrase": {"name": "%s"}}]}} }`, name)
	request, e := http.NewRequest("GET", url, strings.NewReader(doc))
	if e != nil {
		fmt.Println(e.Error())
		return
	}
	request.Header.Set("Content-Type", "application/json")
	resp, e := g_HttpClient.Do(request)
	//resp, e := http.DefaultTransport.RoundTrip(request)
	if e != nil {
		fmt.Println(e.Error())
		return
	}
	defer resp.Body.Close()
	fmt.Println("HEAD ", url, doc)

	// parse
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = json.Unmarshal(body, &result)
	if err != nil {
		fmt.Println(err)
		return
	}
	hitLen := len(result.Hits.Hits)
	if hitLen == 0 {
		fmt.Println("hit ", result.Hits.Total)
		return
	}

	for _, v := range result.Hits.Hits {
		rsp.Timestamp = v.Item.Timestamp
		rsp.Name = v.Item.Name
		rsp.Backend = v.Item.Backend
		rsp.Size = v.Item.Size
		rsp.Hash = v.Item.Hash
		return
	}
	return
}

// ================================= backend =================================
/*
curl -XPUT http://127.0.0.1:9200/backends/_doc/http_127.0.0.1:2021 -H 'Content-Type: application/json' -d '{
    "@timestamp" : "2022-12-31T10:24:01.369Z",
    "backend": "http://127.0.0.1:2021",
	"alive" : 1,
	"policy": 0,
	"magic": 0,
	"size": 0,
	"proxy": 0,
	"weight": 1
}'
*/
func AddBackend(scheme, host string, alive, policy int, magic float32, size, proxy uint64, weight uint8) (e error) {

	url := fmt.Sprintf("%s%s", g_DbURL+"/backends/_doc/", scheme+"_"+host)
	timestr := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	doc := fmt.Sprintf(`{"@timestamp":"%s","backend":"%s://%s","alive":%d,"policy":%d,"magic":%f,"size":%d,"proxy":%d,"weight":%d}`,
		timestr, scheme, host, alive, policy, magic, size, proxy, weight)
	request, e := http.NewRequest("POST", url, strings.NewReader(doc))
	if e != nil {
		fmt.Println(e.Error())
		return
	}
	request.Header.Set("Content-Type", "application/json")

	resp, e := g_HttpClient.Do(request)
	if e != nil {
		fmt.Println(e.Error())
		return
	}
	defer resp.Body.Close()

	return
}

/*
curl -XDELETE localhost:9200/backends/
*/
func DeleteAllBackends() {
	url := fmt.Sprintf("%s", g_DbURL+"/backends/")
	request, e := http.NewRequest("DELETE", url, nil)
	if e != nil {
		fmt.Println(e.Error())
		return
	}
	request.Header.Set("Content-Type", "application/json")
	resp, e := g_HttpClient.Do(request)
	if e != nil {
		fmt.Println(e.Error())
		return
	}
	defer resp.Body.Close()

	return
}

/*
curl -XPOST http://localhost:9200/backends/_doc/http_127.0.0.1:2021/_update -H 'Content-Type: application/json' -d '{
    "doc": {"@timestamp":"2006-01-02T15:04:05.000Z","alive": 1}
}'
*/
func UpdateBackendStatus(scheme, host string, alive uint) (e error) {
	url := fmt.Sprintf("%s%s/_update", g_DbURL+"/backends/_doc/", scheme+"_"+host)
	timestr := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	doc := fmt.Sprintf(`{"doc": {"@timestamp": "%s","alive": %d}}`,
		timestr, alive)
	request, e := http.NewRequest("POST", url, strings.NewReader(doc))
	if e != nil {
		fmt.Println(e.Error())
		return
	}
	request.Header.Set("Content-Type", "application/json")

	resp, e := g_HttpClient.Do(request)
	if e != nil {
		fmt.Println(e.Error())
		return
	}

	defer resp.Body.Close()

	return
}

/*
policy=0 magic = size/weight 容量优先,空间大的节点weight大magic小下次选中几率大
policy=1 magic = proxy/weight代理优先,性能高的节点weight大magic小下次选中几率大

curl -XPOST "http://localhost:9200/backends/_doc/http_127.0.0.1:2021/_update" -H 'Content-Type: application/json' -d'{
  "script" : {"source":"ctx._source.size += 10; ctx._source.proxy += 1; if(ctx._source.weight <=0 ) {ctx._source.weight=1 } if (ctx._source.alive==1) { if (ctx._source.policy==0) {ctx._source.magic =ctx._source.size/ctx._source.weight} if (ctx._source.policy==1) {ctx._source.magic=ctx._source.proxy/ctx._source.weight} }" }
}'
*/
func UpdateBackendStatistics(scheme, host string, size, proxy uint64) (e error) {
	url := fmt.Sprintf("%s%s/_update", g_DbURL+"/backends/_doc/", scheme+"_"+host)
	// 更新统计和magic
	doc := fmt.Sprintf(`{"script" : {"source":"ctx._source.size+=%d;ctx._source.proxy+=%d;if(ctx._source.weight <=0 ) {ctx._source.weight=1 } if (ctx._source.policy==0) {ctx._source.magic=ctx._source.size/ctx._source.weight} if (ctx._source.policy==1) {ctx._source.magic=ctx._source.proxy/ctx._source.weight}"}}`,
		size, proxy)
	request, e := http.NewRequest("POST", url, strings.NewReader(doc))
	if e != nil {
		fmt.Println(e.Error())
		return
	}
	request.Header.Set("Content-Type", "application/json")

	resp, e := g_HttpClient.Do(request)
	if e != nil {
		fmt.Println(e.Error())
		return
	}
	defer resp.Body.Close()

	return
}

/*
curl http://localhost:9200/backends/_search?pretty -H 'Content-Type: application/json' -d '{
	"size": 1,"sort": [{"magic": "asc"}],"query": {"bool": {"must": [ {"match": {"alive": 1}}]}}
}'
*/
func GetNextBackend() (backend string) {
	var result BackendResult
	url := fmt.Sprintf("%s", g_DbURL+"/backends/_search")
	// 在线的backends里找出magic值最小的,达到容量或代理次数的均衡
	doc := fmt.Sprintf(`{ "size": 1,"sort": [{"magic": "asc"}],"query": {"bool": {"must": [ {"match": {"alive": 1}}]}} }`)
	request, e := http.NewRequest("GET", url, strings.NewReader(doc))
	if e != nil {
		fmt.Println(e.Error())
		return
	}
	request.Header.Set("Content-Type", "application/json")
	resp, e := g_HttpClient.Do(request)
	if e != nil {
		fmt.Println(e.Error())
		return
	}
	defer resp.Body.Close()

	// parse
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = json.Unmarshal(body, &result)
	if err != nil {
		fmt.Println(err)
		return
	}
	hitLen := len(result.Hits.Hits)
	if hitLen == 0 {
		fmt.Println("hit ", result.Hits.Total)
		return
	}

	for _, v := range result.Hits.Hits {
		backend = v.Item.Backend
		return
	}
	return
}

/*
curl http://localhost:9200/backends/_search?pretty -H 'Content-Type: application/json' -d '{
	"from": 0,"size": 255,"sort": [{"@timestamp": "desc"}]
}'
*/
func GetAllBackends() (backends []BackendItem) {
	var result BackendResult
	url := fmt.Sprintf("%s", g_DbURL+"/backends/_search")
	doc := fmt.Sprintf(`{ "from": 0,"size": 255,"sort": [{"@timestamp": "desc"}] }`)
	request, e := http.NewRequest("GET", url, strings.NewReader(doc))
	if e != nil {
		fmt.Println(e.Error())
		return
	}
	request.Header.Set("Content-Type", "application/json")
	resp, e := g_HttpClient.Do(request)
	if e != nil {
		fmt.Println(e.Error())
		return
	}
	defer resp.Body.Close()

	// parse
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = json.Unmarshal(body, &result)
	if err != nil {
		fmt.Println(err)
		return
	}
	hitLen := len(result.Hits.Hits)
	if hitLen == 0 {
		fmt.Println("hit 0")
		return
	}

	for _, v := range result.Hits.Hits {
		backends = append(backends, v.Item)
	}
	return
}

/*
curl -XGET http://localhost:9200/objects/_search?pretty -H 'Content-Type: application/json' -d '{
	 "size": 1,"query": {"bool": {"must": [ {"match_phrase": {"name": "/objects/test1"}}]}}
}'
*/
func GetBackendByObject(r *http.Request) (backend string) {
	var result ObjectResult
	url := fmt.Sprintf("%s", g_DbURL+"/objects/_search")
	name := r.URL.Path
	doc := fmt.Sprintf(`{ "size": 1,"query": {"bool": {"must": [ {"match_phrase": {"name": "%s"}}]}} }`, name)
	request, e := http.NewRequest("GET", url, strings.NewReader(doc))
	if e != nil {
		fmt.Println(e.Error())
		return
	}
	request.Header.Set("Content-Type", "application/json")
	resp, e := g_HttpClient.Do(request)
	if e != nil {
		fmt.Println(e.Error())
		return
	}
	defer resp.Body.Close()

	// parse
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = json.Unmarshal(body, &result)
	if err != nil {
		fmt.Println(err)
		return
	}
	hitLen := len(result.Hits.Hits)
	if hitLen == 0 {
		fmt.Println("hit ", result.Hits.Total)
		return
	}

	for _, v := range result.Hits.Hits {
		backend = v.Item.Backend
		return
	}
	return
}
