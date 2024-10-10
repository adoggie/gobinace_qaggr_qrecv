
"""
pip install websocket-client
pip install websocket-client fire pyzmq protobuf -i https://pypi.tuna.tsinghua.edu.cn/simple
sudo apt install libczmq4 -y
sudo apt install libczmq-dev libczmq4 -y
apt install python3-pip
apt install xrdp tmux htop nload iftop nginx sshfs autossh vim stress jq telnet sshfs wget curl expect net-tools nginx tree sshpass gcc iotop sysstat psmisc ntpdate npm mysql-client libmysqlclient-dev pkg-config rsync atop strace lsof nethogs stress network-manager -y
npm install -g pm2 --registry=https://registry.npmmirror.com
ntpdate time.nist.gov
useradd -m -d /home/scott -s /bin/bash scott
"""
import quote_period_pb2
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

os.makedirs('./snap',exist_ok=True)
os.makedirs('./vol',exist_ok=True)

is_zlib = True

# fp = open(f"./vol/BTCUSDT.txt",'w')
fp = open(f"./vol/ETHUSDT.txt",'w')
def on_message(ws, message):
    logger.info('-'*30)
    logger.info(f"Received message: {len(message)}")
    # print(message)
    data = message[18:26]
    #print(len(data))
    fs, = struct.unpack('!q',data)
    post_ts = fs
    data = message[26:]
    data = message
    ts = datetime.datetime.now().timestamp()*1000
    if is_zlib:
        data = zlib.decompress(data)

    pm = quote_period_pb2.PeriodMessage()
    ret = pm.ParseFromString(data)
    post_ts = pm.post_ts
    # print(f"message: {pm}")
    message = pm
    ts = datetime.datetime.now().timestamp()*1000
    logger.info(f"period:{message.period}, ts:{message.ts}, post_ts:{message.post_ts}, poster_id:{message.poster_id} , now:{ts} , ts dealy:{ ts - post_ts}")
    #logger.info(f"period:{message.period}, ts:{message.ts}, post_ts:{message.post_ts}, poster_id:{message.poster_id} , now:{ts} , ts dealy:{ ts - message.post_ts}")
    logger.info(f" message period:{ pm.period}  TS:{ pm.ts}")


    for symbol_info in message.symbol_infos:
        # if symbol_info.symbol == 'ETHUSDT':
        if symbol_info.symbol == 'ETHUSDT':
            inc_bid_count = 0
            inc_ask_count = 0
            for inc in symbol_info.incs:
                inc_bid_count += len(inc.bids)
                inc_ask_count += len(inc.asks)

            logger.info(f"  - symbol:{symbol_info.symbol} trades count:{len(symbol_info.trades)} incs count:{inc_bid_count}/b,{inc_ask_count}/a  snaps count:{len(symbol_info.snaps)}")

            for trade in symbol_info.trades:
                now  = str(datetime.datetime.now()).split(".")[0]
                text = f"{ now} {trade.timestamp} {pm.period} trade_id:{trade.trade_id} price:{trade.price} qty:{trade.qty}"
                fp.write(text + '\n')
                # print(text)
                fp.flush()



def on_error(ws, error):
    print(f"Error: {error}")

def on_close(ws, close_status_code, close_msg):
    print("Connection closed")

def on_open(ws):
    print("Connection opened")
    ws.send("Hello, Server")

# start 提前，unit: second
def run(user='test', start=0, ws_url="127.0.0.1:1501"):
    ws_url = f"ws://{ws_url}/ws?user={user}&start={start}"  # 替换为你的 WebSocket 服务器地址
    ws = websocket.WebSocketApp(
        ws_url,
        on_message=on_message,
        on_error=on_error,
        on_close=on_close
    )
    ws.on_open = on_open
    ws.run_forever()

if __name__ == "__main__":
    fire.Fire()


"""
发送 10s 之前的quote message
python ./ws_client.py run --user='chun' --start=10 --ws_url="127.0.0.1:1501"
发送最新 quote message
python ./ws_client.py run --user='chun' 

python ./ws_client.py run --user='test' --start=0 --ws_url="193.32.149.156:1501"
相同用户 将kickout 前者

python ./ws_client.py run --user='4152cda6' --start=0 --ws_url="45.143.234.189:1501"

python ./ws_client.py run --user='4152cda6' --start=0 --ws_url="185.200.64.182:1501"
专线
python ./ws_client.py run --user='4152cda6' --start=0 --ws_url="203.156.254.197:51501"
专线
python ./ws_client_vol_stat.py run --user='4152cda6' --start=0 --ws_url="203.156.254.197:31501"

python ./ws_client_vol_stat.py run --user='4152cda6' --start=0 --ws_url="185.200.64.182:1501"

统计某一天接收数据总延时
cat ws_client.log | grep '2024-09-03' | grep dealy | awk -F'dealy:' '{sum += $2} END { print sum}'

cat ./vol/ETHUSDT.txt | grep '2024-09-24 10:56' | awk -F ':' '{print $NF}' | awk '{sum += $1} END { print sum}'

"""
