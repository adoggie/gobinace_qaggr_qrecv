import socket
import struct
import time
import datetime


def start_server(host='', port=12345):
    # 创建一个 TCP/IP 套接字
    server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)

    # 绑定套接字到地址
    server_socket.bind((host, port))

    # 启动监听
    server_socket.listen(1)
    print(f"Server started on {host}:{port}")

    while True:
        # 等待客户端连接
        print("waiting for a connection..")
        client_socket, client_address = server_socket.accept()
        print(f"Connection from {client_address}")
        try:
            try:
                while True:
                    # 接收客户端数据
                    time.sleep(3)
                    # data = client_socket.recv(1024)
                    # if not data:
                    #     break
                    hdr = struct.pack('!Q', int(datetime.datetime.now().timestamp()*1000))
                    data = hdr + b'\0'*1024*800
                    # 将数据发送回客户端
                    client_socket.sendall(data)
            finally:
                # 关闭客户端连接
                client_socket.close()
        except Exception as e:
            pass

if __name__ == "__main__":
    start_server()
