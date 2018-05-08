# freedns-go

Optimized DNS Server for Chinese users.

freedns-go uses 2 upstream. One is local DNS upstream like 114, another one is remote DNS upstream like 8.8.8.8. If the results contain any non-China IP or meet some errors, it will use the remote DNS result. This setting is CDN friendly.

The cache policy is Lazy Cache. If there are some querys are expired but it in cache, it will return the old cached value and update it automatically.

## Usage

You can download the prebuilt binary from the [releases](https://github.com/Chenyao2333/freedns-go/releases) page.

```
sudo ./freedns-go -f 114.114.114.114:53 -c 8.8.8.8:53 -l 0.0.0.0:53
```

```
host baidu.com 127.0.0.1
```