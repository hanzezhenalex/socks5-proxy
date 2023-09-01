## socks5-proxy

A proxy server based on socks version 5 protocol.

### Architecture
In the design, we'll have two mode, local and remote. (remote mode is not support yet)<br>

For local mode, only one socks server deployed. The server directly proxies the socks requests.<br>

<img src="https://github.com/hanzezhenalex/socks5-proxy/assets/131222191/4dc94a39-6a50-4fa5-b291-66f4b7a5a74b" width="600px" height="450px">

### How to use

#### Local mode
An easy way is to run our scripts. (docker needed)
```shell
./scripts/deploy.sh
```
The server will be in host network mode using port 1080. Use docker command to check the status.
```shell
docker ps | grep socks5-server
```

### Development guide
details see [here](https://github.com/hanzezhenalex/socks5-proxy/wiki/Development-Guide)

### Reference
SocksV5 RFC: https://www.rfc-editor.org/rfc/rfc1928 <br>
SocksV5 Username/Password Auth: https://www.rfc-editor.org/rfc/rfc1929 <br>
Bind & UDP ASSOCIATE: https://www.jianshu.com/p/55c0259d1a36 <br>
