package main

//type Handler interface {
//		ServeHTTP(ResponseWriter, *Request)
//}

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"plugin"
	"strings"
	"sync"
	"time"
)

type GetRouter func() map[string]http.Handler

func ListDir(dirPth string, suffix string) (files []string, err error) {
	files = make([]string, 0, 10)
	dir, err := ioutil.ReadDir(dirPth)
	if err != nil {
		return nil, err
	}
	PthSep := string(os.PathSeparator)
	suffix = strings.ToUpper(suffix) //忽略后缀匹配的大小写
	for _, fi := range dir {
		if fi.IsDir() { // 忽略目录
			continue
		}
		if strings.HasSuffix(strings.ToUpper(fi.Name()), suffix) { //匹配文件
			files = append(files, dirPth+PthSep+fi.Name())
		}
	}
	return files, nil
}

type PluginHandler struct {
	Path    string
	Plugin  string
	Hash    string
	Handler http.Handler
}

func (p *PluginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h := p.Handler
	if h == nil {
		http.NotFound(w, r)
		return
	}
	h.ServeHTTP(w, r)
}

type PluginHttp struct {
	mu         sync.Mutex
	handlers   map[string]*PluginHandler
	pluginHash map[string]string
}

func (p *PluginHttp) Handle(path string, plugin string, hash string, handler http.Handler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	ph, ok := p.handlers[path]
	if !ok {
		ph = &PluginHandler{Path: path, Plugin: plugin, Hash: hash, Handler: handler}
		p.handlers[path] = ph
		http.Handle(path, ph)
	}
	ph.Path = path
	ph.Plugin = plugin
	ph.Hash = hash
	ph.Handler = handler
}
func (p *PluginHttp) GetHash(path string) string {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", md5.Sum(bytes))
}

func (p *PluginHttp) Load(pluginPath string) {
	defer func() {
		err := recover()
		log.Printf("defer load %s", err)
		if err != nil {
			log.Printf("load error: %s", err)
		}
	}()
	hash := p.GetHash(pluginPath)
	if hash == "" {
		return
	}
	if p.pluginHash[pluginPath] == hash {
		return
	}
	p.pluginHash[pluginPath] = hash
	//is update
	//TODO 当前golang1.8 版本不会更新,并且，注意，插件不能重命名重加载
	plu, err := plugin.Open(pluginPath)
	if err != nil {
		log.Printf("error %s %s", pluginPath, err)
		return
	}
	getRouter, err := plu.Lookup("GetRouter")
	if err != nil {
		log.Printf("error %s %s", pluginPath, err)
		return
	}
	getRouterFunc, ok := getRouter.(func() map[string]http.Handler)
	if !ok {
		log.Printf("error %s %#T function is not GetRouter", pluginPath, getRouter)
		return
	}
	router := getRouterFunc()
	for p, h := range router {
		log.Printf("load %s %s", p, hash)
		pluginHttp.Handle(p, pluginPath, hash, h)
		//http.Handle(p, h)
	}

}

func (p *PluginHttp) UpdatePlugin() {
	plugins, err := ListDir("./plugin/", ".so")
	if err != nil {
		log.Printf("list plugin error:%s", err)
		return
	}
	for _, pluginPath := range plugins {
		log.Printf("load %s file", pluginPath)
		go p.Load(pluginPath)
	}
}

var pluginHttp = &PluginHttp{handlers: make(map[string]*PluginHandler), pluginHash: make(map[string]string)}

func main() {
	pluginHttp.UpdatePlugin()
	http.ListenAndServe(":7001", nil)
}
