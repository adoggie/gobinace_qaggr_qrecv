import websockets
import threading
import asyncio
import time
import struct
import datetime

async def send_large_data(websocket):
    while True:
        # 生成400KB的数据
        data = b'x' * 400 * 1024
        hdr = struct.pack('!p', datetime.datetime.now().timestamp()*1000)
        data = hdr + b'\0'*1024*400

        await websocket.send(data)
        await asyncio.sleep(3)

async def main():
    async with websockets.serve(send_large_data, "", 12345):
        await asyncio.Future()  # Run forever

# 使用线程来运行 asyncio 事件循环
def run_server():
    asyncio.run(main())

if __name__ == "__main__":
    server_thread = threading.Thread(target=run_server)
    server_thread.start()
