package main

const (
	URLS    = "$URL: version.go "
	AUTHOR  = "$Author: scott "
	VERSION = "0.3.6 - $Revision: 3419 $ "
	DATE    = "$Date: 2024-10-07 20:45:37 +0800 (Mon, 07 Oct 2024) $  "
)

/**
svn menu add followings:
 svn propset svn:keywords "Id Date" index.php

0.3.6
  - fixed : 修复 send_hdr 功能问题

0.3.5
 - fixed: wsserver.send()  取消 append() 低效率（大概率）

0.3.4
 - fixed ： controller.go 发送前错误修改 periodMessage.Ts = time.Now().Unix ,删除

0.3.3
  - controller.go 增加空行情的超时检查，超过20s 接收不到行情数据，发送告警提示

0.3.2
  - added: 空period数据检查，不再发送 ，发送告警提示，连续发送将延时5s
  - added: 启动时，定时上报运行状态信息
  - added: core.qmon.go 服务监控模块 发送到zmq

0.3.1
  1. fixed : aggregation.go
     错误将 Asks Inc 加入了Bids

0.3.0 features
  1. 增加snapshot 消息接收
  2. 增加 put_snapshot  ,
  3. timeitr() ->detach 过程增加 snapshot 数据处理
  4. 增加控制变量 time_iterate_interval

0.2.8 features: $Revision: 3419 $
 1. fixed:  param start 计算偏差

0.2.7 features: $Revision: 3343
 1. websocket server send 增加 26 bytes hdr

0.2.6  features: Revision: 3341
 1. 取消 UpdateTradingSymbolInterval 定时更新交易品种
 2. period 与BornTs 根据ORIGIN_UTC_TIME ( 2024/01/01 00:00:00) 偏移量计算获得（1704067200)

0.2.5  features:
 1. 增加 定时更新交易品种 UpdateTradingSymbolInterval ，默认: 6 hours

0.2.4  features:
 - websocket server：
    进入发送队列时将 periodMessage 进行压缩，对于缓存指定时间的行情数据可以 1:3 的压缩比例
    增加: WsSendData 数据结构，用于压缩和缓存，替换直接缓存 * message.PeriodMessage

0.2.3  features:
 - add： 增加websocket 分发功能
    1. users.json 动态用户账号添加
    2. 多用户连接，每个用户独立数据流
	3. fixed much bugs
	4. 测试 start 提前行情
 		发送 10s 之前的quote message
		发送最新 quote message
		相同用户 将kickout 前者


0.2.2  features:
 - add： 每日定时读取新上架交易币，创建SymbolDef结构用于缓存和分发交易数据


0.2.1   $Revision: 3419 $
 - 重新梳理put_xxx()的处理过程，对于过早到达的数据丢弃，对于最新过时的数据重新利用
 - [p1/period1,p2/period2] 此刻来了p3数据在之前处理时将丢弃，现在保留在pending队列中，等待时钟走到p3时从pending中取出
 - < p1 数据到达时，合并到p1 返回

0.2.0  2024.08.20 $Revision: 3419 $
 - 取消 kafka 之前的压缩，采用kafka内建的snappy 压缩算法
 - kafka 采用 github.com/segmentio/kafka-go 发送数据( key / value 尚未搞明白)
 - zmq 改用 github.com/pebbe/zmq4
 - 观察 aggr进程的内存是否连续增长
 - pb : message.TS /Period 字段填充已修改

0.1.0 - $Revision:
 - kafka 压缩zlib数据之后发送
 - python client 读取zlib数据解压，还原打印
 - zmq 发送/接收出现失败情况，即便是sndhw 改为0 也会出现
 - 实现从 qrs -> qas -> kafka -> client 数据链路导通
*/
