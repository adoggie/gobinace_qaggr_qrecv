package main

const (
	URLS    = "$URL: version.go "
	AUTHOR  = "$Author: scott "
	VERSION = "0.5.1 - $Revision: 3403 $ "
	DATE    = "$Date: 2024-09-24 11:57:11 +0800 (Tue, 24 Sep 2024) $  "
)

/**
0.5.1
  - 修复接收trade 从spot源的问题，在 pkg/go-binance-2.5.0/v2/ features代码中添加 接收 trade

0.5.0  Revision: 3400
  - fixed : 连接失败之后等待重连 sleep / continue

0.4.0  Revision: 3394
  - added:  market_aggtrade.go
            market_kline.go
0.3.4 3387
  - 修改 WsDepthServe 连接代码
  - 修改 数据编码支持 gob ，pb 两种格式
  - zmq 发布采用 frame1+fame2 格式

0.3.3  r:3383
  - WsCombinedDiffDepthServe 订阅行情改为100ms
0.3.2
  - added: 启动时，定时上报运行状态信息


0.3.1
  1.  snapshot add "request_wait " 用于控制请求等待时间，防止ip 被 banned ，请求太快

0.3.0
  - 增加snapshot
   1. inc 接收重连时，发送snapshot
   2. snapmgr 定时发送snapshot
   3. inc / trade 接收 service 发现连接上订阅的symbol 在指定时间内均无最新消息，则断开连接重连
       RecvTradeReconnectTimeout , RecvOrbIncReconnectTimeout=120
   4. inc 检查发现不连续时 发送 snapshot ( 暂时不支持)

0.1.0 - $Revision:
 - kafka 压缩zlib数据之后发送
 - python client 读取zlib数据解压，还原打印
 - zmq 发送/接收出现失败情况，即便是sndhw 改为0 也会出现
 - 实现从 qrs -> qas -> kafka -> client 数据链路导通
*/
