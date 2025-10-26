package db

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/lithammer/shortuuid/v4"
)

func NewRemoteRessourceDB() *RemoteRessourceDB {
	return &RemoteRessourceDB{
		idToRessource: make(map[string]*RemoteRessource),
		mu:            sync.RWMutex{},
	}
}

type RemoteRessource struct {
	ID      string                  `json:"id"`
	URL     *SafeURL                `json:"url"`
	RuleSet *RemoteRessourceRuleSet `json:"rule_set,omitempty"`
}

type SafeURL struct {
	urlB64 string
}

func NewSafeURL(us string) (*SafeURL, error) {
	//base64 encode the url
	// does it parse as proper url ?
	_, err := url.Parse(us)
	if err != nil {
		return nil, err
	}
	urlB64 := base64.StdEncoding.EncodeToString([]byte(us))
	return &SafeURL{urlB64: urlB64}, nil
}

func (s *SafeURL) URL() string {
	// ret
	decoded, err := base64.StdEncoding.DecodeString(s.urlB64)
	if err != nil {
		return ""
	}
	return string(decoded)
}

func (s *SafeURL) String() string {
	return s.urlB64
}

type RemoteRessourceDB struct {
	idToRessource map[string]*RemoteRessource
	mu            sync.RWMutex
}

// hex digest
func hashURL(us string) string {
	hash := sha256.New()
	hash.Write([]byte(us))
	return hex.EncodeToString(hash.Sum(nil))
}

func MakeProxyRessource(us string) (*ProxyRessource, error) {
	parsedURL, err := url.Parse(us)
	if err != nil {
		return nil, err
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, fmt.Errorf("invalid url")
	}

	hash := hashURL(us)
	safeUrl, err := NewSafeURL(us)
	if err != nil {
		return nil, err
	}
	return &ProxyRessource{url: safeUrl, ID: hash}, nil
}

func (r *RemoteRessourceDB) GetOrSet(us string, ruleSet *RemoteRessourceRuleSet, override bool) (*ProxyRessource, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// is and id of existing?
	res, ok := r.idToRessource[us]
	if ok {
		if !override {
			return &ProxyRessource{url: res.URL, ID: us, RuleSet: res.RuleSet}, nil
		}
	}

	fmt.Println("Getting or setting remote ressource: ", us)

	// parses as proper url ?
	parsedURL, err := url.Parse(us)
	if err != nil {
		return nil, err
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, fmt.Errorf("invalid url")
	}

	hash := hashURL(us)
	// already exists in known ressources?
	res, ok = r.idToRessource[hash]
	if ok {
		if !override {
			return &ProxyRessource{url: res.URL, ID: hash, RuleSet: res.RuleSet}, nil
		}
	}

	safeUrl, err := NewSafeURL(us)
	if err != nil {
		return nil, err
	}

	r.idToRessource[hash] = &RemoteRessource{ID: hash, URL: safeUrl, RuleSet: ruleSet}
	return &ProxyRessource{url: safeUrl, ID: hash}, nil
}

type ProxyRessource struct {
	url     *SafeURL
	ID      string                  `json:"id"`
	RuleSet *RemoteRessourceRuleSet `json:"rule_set,omitempty"`
}

func (p *ProxyRessource) URL() string {
	return p.url.URL()
}

type FusibleRessource struct {
	ID             string          `json:"id"`
	ProxyRessource *ProxyRessource `json:"proxy_ressource"`
	melted         sync.Once
	RuleSet        *FusibleRessourceRuleSet `json:"rule_set,omitempty"`
}

type FusibleRessourceDB struct {
	ressources map[string]*FusibleRessource
	mu         sync.RWMutex
}

func NewFusibleRessourceDB() *FusibleRessourceDB {
	return &FusibleRessourceDB{
		ressources: make(map[string]*FusibleRessource),
		mu:         sync.RWMutex{},
	}
}

func (f *FusibleRessourceDB) MakeFusibleRessource(ProxyRessource *ProxyRessource) (*FusibleRessource, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var ruleSet *FusibleRessourceRuleSet
	if ProxyRessource.RuleSet != nil {
		if ProxyRessource.RuleSet.FusibleRessourceRuleSet != nil {
			ruleSet = ProxyRessource.RuleSet.FusibleRessourceRuleSet
		}
	}
	newFusible := &FusibleRessource{ID: shortuuid.New(), ProxyRessource: ProxyRessource, RuleSet: ruleSet}
	f.ressources[newFusible.ID] = newFusible
	return newFusible, nil
}

func (f *FusibleRessourceDB) ConsumeFusibleRessource(id string) (*ProxyRessource, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	// exists ?
	fusible, ok := f.ressources[id]
	if !ok {
		return nil, fmt.Errorf("fusible ressource not found")
	}
	// consume
	fusible.melted.Do(func() {
		delete(f.ressources, id)
	})
	return &ProxyRessource{url: fusible.ProxyRessource.url, ID: fusible.ProxyRessource.ID}, nil
}

func ProxyRessourceFromFuseID(db *FusibleRessourceDB, id string) (*ProxyRessource, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.ConsumeFusibleRessource(id)
}

func FusibleRessourceFromRessourceID(fdb *FusibleRessourceDB, rdb *RemoteRessourceDB, rid string, req *http.Request) (*FusibleRessource, error) {
	rdb.mu.RLock()
	defer rdb.mu.RUnlock()
	remote, ok := rdb.idToRessource[rid]
	if !ok {
		return nil, fmt.Errorf("remote ressource not found")
	}
	if remote.RuleSet != nil {
		if !remote.RuleSet.Validate(req, RequestTypeRequest) {
			return nil, fmt.Errorf("remote ressource rule set not valid")
		}
	}
	return fdb.MakeFusibleRessource(&ProxyRessource{url: remote.URL, ID: rid, RuleSet: remote.RuleSet})
}

func UrlToFusibleRessource(fdb *FusibleRessourceDB, url string, ruleSet *RemoteRessourceRuleSet) (*FusibleRessource, error) {
	proxyRessource, err := MakeProxyRessource(url)
	if err != nil {
		return nil, err
	}
	return fdb.MakeFusibleRessource(&ProxyRessource{url: proxyRessource.url, ID: proxyRessource.ID, RuleSet: ruleSet})
}
