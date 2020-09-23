package freedns

import (
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/miekg/dns"
)

type upstreamProvider interface {
	GetUpstream() string
}

type staticUpstreamProvider struct {
	upstream string
}

func (provider *staticUpstreamProvider) GetUpstream() string {
	return provider.upstream
}

type resolvconfUpstreamProvider struct {
	filename string
	// keep last valid servers even if file becomes invalid
	servers      []string
	serversMutex sync.RWMutex
}

func (provider *resolvconfUpstreamProvider) GetUpstream() string {
	provider.serversMutex.RLock()
	defer provider.serversMutex.RUnlock()
	// Always use the first one for now
	return provider.servers[0]
}

func parseServersFromResolvconf(filename string) ([]string, error) {
	parsedConfig, err := dns.ClientConfigFromFile(filename)
	if err != nil {
		return nil, err
	}

	servers := make([]string, 0)
	for _, addr := range parsedConfig.Servers {
		if normalized, err := normalizeDnsAddress(addr); err == nil {
			servers = append(servers, normalized)
		}
	}

	if len(servers) == 0 {
		return nil, Error("No servers found in " + filename)
	}

	return servers, nil
}

func newResolvconfUpstreamProvider(filename string) (*resolvconfUpstreamProvider, error) {
	servers, err := parseServersFromResolvconf(filename)
	if err != nil {
		return nil, err
	}

	provider := &resolvconfUpstreamProvider{
		filename: filename,
		servers:  servers,
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := watcher.Add(filename); err != nil {
		return nil, err
	}
	go func() {
		// TODO: `watcher` will outlive upstream provider
		// we need to add a Close method to upstreamProvider
		defer watcher.Close()
		logger := log.WithField("filename", filename)
		logger.Info("Start watching")
		for {
			select {
			case _, ok := <-watcher.Events:
				if !ok {
					logger.Warn("Watch failed")
					return
				}
			case err, ok := <-watcher.Errors:
				logger.WithField("error", err).WithField("ok", ok).Warn("Watch failed")
				return
			}
			logger.Info("Reload resolvconf")

			servers, err := parseServersFromResolvconf(filename)
			if err != nil {
				logger.WithField("error", err).Warn("Cannot read servers, ignore")
				continue
			}

			provider.serversMutex.Lock()
			provider.servers = servers
			provider.serversMutex.Unlock()
		}
	}()

	return provider, nil
}

// Create upstream provider based on upstream name
//
// Possible name values are:
// IP address (with optional port) :: use this IP as static upstream
// Filename :: parse the file as resolv.conf, read upstreams from the file (monitor file change)
func newUpstreamProvider(name string) (upstreamProvider, error) {
	if addr, err := normalizeDnsAddress(name); err == nil {
		return &staticUpstreamProvider{
			upstream: addr,
		}, nil
	}
	if fileinfo, err := os.Stat(name); err == nil && !fileinfo.IsDir() {
		return newResolvconfUpstreamProvider(name)
	}
	return nil, Error("Invalid upstream name " + name)
}
