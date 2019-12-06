# freedns-go

Optimized DNS Server for Chinese users.

freedns-go tries to dispatch the request to the local (maybe poisoned, set by `-f`) DNS prioritly. If it detected any non-Chinese websites, it fallbacks to dispatch the request to the remote (trustable, set by `-c`) DNS server.

The cache policy is Lazy Cache. If there are some records are expired but in the cache, it will return the cached records and update it asynchronously.

## Usage

You can download the prebuilt binary from the [releases](https://github.com/Chenyao2333/freedns-go/releases) page.

```
sudo ./freedns-go -f 114.114.114.114:53 -c 8.8.8.8:53 -l 0.0.0.0:53
```

```
host baidu.com 127.0.0.1
```

![](https://pppublic.oss-cn-beijing.aliyuncs.com/pics/%E5%B1%8F%E5%B9%95%E5%BF%AB%E7%85%A7%202018-05-08%20%E4%B8%8B%E5%8D%889.49.36.png)

freedns-go just dispatches your queries to the optimal upstreams. Your network should be able to reach those upstreams.s (e.g. 8.8.8.8). You can do that by port forwarding, or anyways you like..**
