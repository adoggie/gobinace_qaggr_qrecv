package core

const (
	URLS    = "$URL: svn://bzz.wallizard.com:4000/EL/branches/goCCSuites2024/core/version.go $"
	AUTHOR  = "$Author: scott $"
	VERSION = "0.6.4 - $Revision: 3168 $"
	DATE    = "$Date: 2024-05-24 21:37:15 +0800 (Fri, 24 May 2024) $ "
)

/**
0.6.4
1.swap 撤单之后再次报单，在统计回报中的swapdollar金额数量只有最后一次成交的金额

** 问题定位：
> swap报单之后产生撤单再报单，之前撤单的动作最后没有 orderFilled 返回，返回orderCancel ， 所以在最后统计的时候只有最有一个filled order记录被计算在内 。
> 需要 统计 order返回的所有 partialFilled ，filled记录。


0.6.3
1. 修改 duijiaratio "ordermgr.go:FormatTaskInfo() 计算参数的位置
2. CFX-USDT 开合约计算数量时(firstOrder)采用了 spot的ask价格，得出的合约数量*合约价格超出了 报单任务的金额（ spot 与 swap 价差偏大 才体现问题)

0.6.1
1. ws 发送委托改成 rest发送委托，捕捉一下发送失败的错误信息

*/
