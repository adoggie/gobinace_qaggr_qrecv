import websockets
import asyncio

async def receive_data():
    async with websockets.connect("ws://10.120.30.6:12345") as websocket:
        while True:
            data = await websocket.recv()
            print(f"Received {len(data)} bytes")

asyncio.run(receive_data())
