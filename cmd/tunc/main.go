package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/4396/tun/client"
	"github.com/4396/tun/log"
	"github.com/4396/tun/log/impl"
	"github.com/4396/tun/version"
	"gopkg.in/ini.v1"
)

var (
	conf   = flag.String("c", "tunc.ini", "config file's path")
	server = flag.String("server", "", "tun server addr")
	id     = flag.String("id", "", "tun proxy id")
	addr   = flag.String("addr", "", "local server addr")
	token  = flag.String("token", "", "tun proxy token")
)

type proxy struct {
	Addr  string
	Token string
}

type config struct {
	Server  string
	Proxies map[string]*proxy
}

func parse(filename string, cfg *config) (err error) {
	_, errSt := os.Stat(*conf)
	if errSt != nil {
		return
	}

	f, err := ini.Load(filename)
	if err != nil {
		return
	}

	for _, sec := range f.Sections() {
		id := sec.Name()
		if id == "tunc" {
			cfg.Server = sec.Key("server").String()
			continue
		}

		token := sec.Key("token").String()
		if token == "" {
			continue
		}

		addr := sec.Key("addr").String()
		if addr == "" {
			continue
		}

		cfg.Proxies[id] = &proxy{
			Addr:  addr,
			Token: token,
		}
	}
	return
}

func loadConfig() (cfg *config, err error) {
	cfg = &config{
		Proxies: make(map[string]*proxy),
	}

	err = parse(*conf, cfg)
	if err != nil {
		return
	}

	if *server != "" {
		cfg.Server = *server
	}

	if *id != "" && *addr != "" {
		cfg.Proxies[*id] = &proxy{
			Addr:  *addr,
			Token: *token,
		}
	}
	return
}

func main() {
	flag.Parse()
	log.Use(&impl.Logger{})
	log.Infof("start tun client, version is %s", version.Version)

	cfg, err := loadConfig()
	if err != nil {
		log.Errorf("failed to load configuration file, err=%v", err)
		return
	}

	var (
		count int64
		first = true
		ctx   = context.Background()
	)

LOOP:
	if first {
		first = false
	} else {
		count++
		time.Sleep(time.Second)
		log.Infof("%d times reconnect to tun server", count)
	}

	c, err := client.Dial(cfg.Server)
	if err != nil {
		goto LOOP
	}

	count = 0
	log.Info("connect to tun server success")

	for id, proxy := range cfg.Proxies {
		err = c.Proxy(id, proxy.Token, proxy.Addr)
		if err != nil {
			if err == client.ErrClosed {
				goto LOOP
			}
			log.Errorf("failed to load proxy, err=%v, id=%s, addr=%s", err, id, proxy.Addr)
			return
		}
		log.Infof("load proxy success, id=%s, addr=%s", id, proxy.Addr)
	}

	err = c.Run(ctx)
	if err != nil {
		if err == client.ErrClosed {
			goto LOOP
		}
		log.Errorf("failed to run tun client, err=%v", err)
	}
}
