#coding:utf-8

import os,os.path,sys,time,datetime,traceback,json
import zmq
import fire
import base64
import json
def init_keepalive(sock):
    sock.setsockopt(zmq.TCP_KEEPALIVE,1)
    sock.setsockopt(zmq.TCP_KEEPALIVE_IDLE,120)
    sock.setsockopt(zmq.TCP_KEEPALIVE_INTVL,1)
    sock.set_hwm(0)

MX_PUB_ADDR="tcp://127.0.0.1:1608"
MX_SUB_ADDR="tcp://127.0.0.1:1609"

def do_sub(sub_addr=MX_SUB_ADDR,decode=False,topic=''):
    ctx = zmq.Context()
    sock = ctx.socket(zmq.SUB)
    init_keepalive(sock)
    sock.setsockopt(zmq.SUBSCRIBE, b'')  # 订阅所有品种
    sock.connect(sub_addr)
    while True:
        frames = sock.recv_multipart()
        print(frames)


def do_sub1(sub_addr=MX_SUB_ADDR):
    ctx = zmq.Context()
    sock = ctx.socket(zmq.SUB)
    init_keepalive(sock)
    sock.setsockopt(zmq.SUBSCRIBE, b'')  # 订阅所有品种
    sock.connect(sub_addr)
    while True:
        m = sock.recv()
        fs = m.decode().split(',')
        if fs[0] != '1.0':
            continue
        ts = datetime.datetime.fromtimestamp( float(fs[6])/1000)
        print(m,str(ts))

def do_sub2(sub_addr=MX_SUB_ADDR):
    ctx = zmq.Context()
    sock = ctx.socket(zmq.SUB)
    init_keepalive(sock)
    sock.setsockopt(zmq.SUBSCRIBE, b'')  # 订阅所有品种
    sock.connect(sub_addr)
    while True:
        m = sock.recv()
        print(str(datetime.datetime.now()),m)

def do_pub(text,pub_addr=MX_PUB_ADDR):
    ctx = zmq.Context()
    sock = ctx.socket(zmq.PUB)
    init_keepalive(sock)
    sock.connect(pub_addr)
    time.sleep(.5)  # 必须等一下，zmq bug
    for n in range(100000):
        t = f"{n}.{text}"
        sock.send(t.encode())
        print("msg sent:",t)
        # sock.close()
        time.sleep(1)

    sock.close()

if __name__ == '__main__':
    fire.Fire()