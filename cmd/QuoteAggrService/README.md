### QuoteAggrService

多路行情接收聚合之后分发 -> zmq / kafka

### install 
two agg install : 
  quoteRecv 配置两个接收agg
  nginx 配置反向代理到两个agg

### nginx proxy 
apt install libnginx-mod-stream
nginx -t 
nginx -s reload 
```shell
events {
        worker_connections 768;
        # multi_accept on;
}

k

```

### pm2
pm2 start -n qrecv1 ./qrecv_0.2.8 --cron-restart="30 8,21 * * * "
