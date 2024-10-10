
"""
pip install websocket-client fire pyzmq protobuf -i https://pypi.tuna.tsinghua.edu.cn/simple
sudo apt install libczmq4 -y
sudo apt install libczmq-dev libczmq4 -y
apt install python3-pip
apt install xrdp tmux htop nload iftop nginx sshfs autossh vim stress jq telnet sshfs wget curl expect net-tools nginx tree sshpass gcc iotop sysstat psmisc ntpdate npm mysql-client libmysqlclient-dev pkg-config rsync atop strace lsof nethogs stress network-manager -y
npm install -g pm2 --registry=https://registry.npmmirror.com
ntpdate time.nist.gov
useradd -m -d /home/scott -s /bin/bash scott

启动 qrecv_0.4.0
配置：

[depth_client_manager]
depth_level = 10

[kline_manager]
interval = "1m"

[aggtrade_manager]
reconnect_wait_time = 5

recv_kline_enable = false
recv_aggtrade_enable = false
recv_depth_enable = true
data_encode_type = 'gob' # pb

fanout_servers = ["tcp://127.0.0.1:15541"]

"""
import quote_period_pb2
import quote_common_pb2
import websocket
import datetime
import zlib
import fire
import os,sys,shutil
import logging
import traceback
import struct
from logging.handlers import TimedRotatingFileHandler
import numpy as np

import zmq
import fire


def init_keepalive(sock):
    sock.setsockopt(zmq.TCP_KEEPALIVE,1)
    sock.setsockopt(zmq.TCP_KEEPALIVE_IDLE,120)
    sock.setsockopt(zmq.TCP_KEEPALIVE_INTVL,1)
    sock.set_hwm(0)


def init_logger(log_level="DEBUG", log_path=None, logfile_name=None,stdout=False, clear=False, backup_count=0):
    """ 初始化日志输出
    @param log_level 日志级别 DEBUG/INFO
    @param log_path 日志输出路径
    @param logfile_name 日志文件名
    @param clear 初始化的时候，是否清理之前的日志文件
    @param backup_count 保存按天分割的日志文件个数，默认0为永久保存所有日志文件
    """
    logger = logging.getLogger()
    logger.setLevel(log_level)
    fmt_str = "%(levelname)1.1s [%(asctime)s] %(message)s"
    fmt = logging.Formatter(fmt=fmt_str, datefmt=None)

    if logfile_name:
        if clear and os.path.isdir(log_path):
            shutil.rmtree(log_path)
        if not os.path.isdir(log_path):
            os.makedirs(log_path)
        logfile = os.path.join(log_path, logfile_name)
        handler = TimedRotatingFileHandler(logfile, "midnight", backupCount=backup_count)
        handler.setFormatter(fmt)
        logger.addHandler(handler)
        print("init logger ...", logfile)
    # else:
    if stdout:
        print("init logger ... Console Stream")
        handler = logging.StreamHandler()
        handler.setFormatter(fmt)
        logger.addHandler(handler)
    return logger

logger = init_logger(log_level="DEBUG", log_path="./log", logfile_name="ws_client.log", stdout=True, clear=False, backup_count=0)

os.makedirs('./zmq_recv_log',exist_ok=True)
os.makedirs('./vol',exist_ok=True)
fp = open(f"./vol/ETHUSDT.txt",'w')

is_zlib = True

MX_SUB_ADDR="tcp://127.0.0.1:15541"

def do_sub(sub_addr=MX_SUB_ADDR,topic=''):
    ctx = zmq.Context()
    sock = ctx.socket(zmq.SUB)
    init_keepalive(sock)
    sock.setsockopt(zmq.SUBSCRIBE, topic.encode())  # 订阅所有品种
    # sock.connect(sub_addr)
    sock.bind(sub_addr)
    print("Bind to:",sub_addr)
    while True:
        frames = sock.recv_multipart()
        # print(frames)
        if len(frames) != 2:
            print( datetime.datetime.now(),"Invalid ZMq Frames:",len(frames[0]))
            continue
        # print('Topic:',frames[0])
        # print("Body Size:",len(frames[1]))
        topic = frames[0].decode()
        family,symbol = topic.split('/')
        data = frames[1]
        if family == 'depth':
            message = quote_common_pb2.DepthInfo()

        if family == 'aggtrade':
            message = quote_common_pb2.AggTradeInfo()

        if family == 'kline':
            message = quote_common_pb2.KlineInfo()


        if family == 'snapshot':
            message = quote_period_pb2.SnapshotInfo()

        if family == 'inc':
            message = quote_period_pb2.IncrementOrderBookInfo()


        if family == 'trade':
            message = quote_period_pb2.TradeInfo()

        message.ParseFromString(data)

        if symbol == 'ETHUSDT':
            if family == 'depth':
                # print("Depth:")
                # print(message)
                pass
            if family == 'aggtrade':
                # print("AggTrade:")
                # print(message)
                pass
            if family == 'kline':
                # print("Kline:")
                # print(message)
                pass
            if family == 'trade':
                # print("Trade:")
                # print(message)

                trade = message
                now  = str(datetime.datetime.now()).split(".")[0]
                text = f"{ now} {trade.timestamp} trade_id:{trade.trade_id} price:{trade.price} qty:{trade.qty}"
                fp.write(text + '\n')
                print(text)
                fp.flush()
                pass







if __name__ == "__main__":
    fire.Fire()


"""
python ./zmq_recv_client.py do_sub 

cat ./vol/ETHUSDT.txt | grep '2024-09-24 10:56' | awk -F ':' '{print $NF}' | awk '{sum += $1} END { print sum}'
"""
