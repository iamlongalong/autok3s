package airgap

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	defaultClient = http.Client{
		Timeout: 45 * time.Second,
	}

	dfClientHub = clientHub{}
)

type clientHub struct {
	clients []struct {
		c http.Client
		m matcher
	}
}

func (ch *clientHub) Set(m matcher, c http.Client) {
	ch.clients = append(ch.clients, struct {
		c http.Client
		m matcher
	}{c: c, m: m})
}

func (ch *clientHub) Get(urlstr string) http.Client {
	u, err := url.Parse(urlstr)
	if err != nil {
		return defaultClient
	}

	for _, client := range ch.clients {
		if client.m.Match(u.Host) {
			logrus.Infof("matched domain: %s", urlstr)
			return client.c
		}
	}

	return defaultClient
}

type matcher interface {
	Match(string) bool
}

type DomainMatcher struct {
	Domains []string
}

func (dm *DomainMatcher) Match(urlstr string) bool {
	u, err := url.Parse(urlstr)
	if err != nil {
		return false
	}
	host := u.Hostname()

	for _, domain := range dm.Domains {
		if strings.HasSuffix(host, domain) {
			return true
		}
	}

	return false
}

func init() {
	// domain:proxy,domain:proxy
	proxyStrLists := os.Getenv("AUTOK3S_PROXY")
	for _, proxyMatchersStr := range strings.Split(proxyStrLists, ",") {
		if strings.TrimSpace(proxyMatchersStr) == "" {
			continue
		}

		proxyStrSlis := strings.SplitN(proxyMatchersStr, ":", 2)
		if len(proxyStrSlis) != 2 {
			logrus.Warn(fmt.Sprintf("invalid proxy matcher format: %s", proxyMatchersStr))
			continue
		}

		proxyStr, domainStr := proxyStrSlis[0], proxyStrSlis[1]
		if strings.TrimSpace(proxyStr) == "" || strings.TrimSpace(domainStr) == "" {
			logrus.Warn(fmt.Sprintf("invalid proxy matcher format: %s", proxyMatchersStr))
			continue
		}

		proxyURL, err := url.Parse(proxyStr)
		if err != nil {
			logrus.Warnf("Parse proxy url %s failed: %v", proxyStr, err)
			continue
		}

		domainURL, err := url.Parse(domainStr)
		if err != nil {
			logrus.Warnf("Parse domain url %s failed: %v", domainStr, err)
			continue
		}

		dfClientHub.Set(&DomainMatcher{Domains: []string{domainURL.Host}}, http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			},
		})
	}
}
