import socket
import struct
import datetime


# def start_client(host='10.120.30.6', port=12345):
def start_client(host='193.32.149.156', port=12345):
    # 创建一个 TCP/IP 套接字
    client_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)

    # 连接到服务器
    client_socket.connect((host, port))
    body = b''
    try:
        while True:
            # 获取用户输入

            # 发送数据到服务器
            payload_size = 1024*800 + 8

            res = client_socket.recv(1024*1024*10)
            body += res
            # print(f"body size: {len(res)}")
            if len(body) >= payload_size:
                res = body[:payload_size]
                body = body[payload_size:]

                hdr = res[:8]
                ts0,=struct.unpack('!q',hdr)
                ts1 = datetime.datetime.now().timestamp()*1000
                print(f"msgsize:{len(res)} delay: {ts1-ts0}")


    finally:
        # 关闭套接字
        client_socket.close()

if __name__ == "__main__":
    start_client()
