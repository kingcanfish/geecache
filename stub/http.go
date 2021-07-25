package geecache

import (
	"fmt"
	"github.com/kingcanfish/geecache/consistenthash"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/_geecache/"
	defaultReplicas = 50
)

// HTTPPool implements PeerPicker for a pool of HTTP peers.
type HTTPPool struct {
	//self 用来记录自己的地址 包括主机名和端口名
	self string
	// basePath 作为通讯节点的前缀
	basePath string
	mu       sync.Mutex
	// peers 用来根据具体的 key 来选择对应的节点
	peers *consistenthash.Map
	// 映射远程节点对应的 httpGetter
	//每一个远程节点都有一个 HTTPGetter 的实例去请求他
	HTTPGetters map[string]*HTTPGetter
}

// NewHTTPPool 用来生成一个新 HTTPPool 实例
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// Log 打印带有 server name 的日志信息
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("http pool unexpected path:" + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)

	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}

// Set 设定更新 Pool 的 peer 列表
// 方法实例化了一致性哈希算法，并且添加了传入的节点。
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.HTTPGetters = make(map[string]*HTTPGetter, len(peers))
	for _, peer := range peers {
		p.HTTPGetters[peer] = &HTTPGetter{baseURL: peer + p.basePath}
	}
}

// PickPeer 更具 key 来选择对应的 peer
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.HTTPGetters[peer], true
	}
	return nil, false
}

// HTTPGetter 是请求远端节点用的客户端
// 每一个远端节点会对应一个 客户端实例
type HTTPGetter struct {
	baseURL string
}

func (h *HTTPGetter) Get(group string, key string) ([]byte, error) {
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key),
	)
	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}
	return bytes, nil
}

var _ PeerGetter = (*HTTPGetter)(nil)
