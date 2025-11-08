import asyncio
import os

# from sdk2 import SDK
from onql_ext_sdk import SDK

import protocol
import request_handler

# async def main():
#     await sdk.init("schema", nats_url="nats://localhost:4222")

# # main()
# asyncio.run(main())


async def main():
    sdk = await SDK.create("protocol", nats_url="nats://localhost:4222")
    sc = protocol.Protocol(sdk)

    # table = schema.Table()
    # table.add("name", "string", "disk", "no")
    # table.add("age", "number", "disk", "no", "")
    # table.add("email", "string", "disk", "no", "")
    # table.add("password", "string", "disk", "yes", "")

    # for i in range(1):
    #     # await sc.createDatabase(f"test")
    #     await sc.createTable("test", f"users", table.get())
    rh = request_handler.RequestHandler(sc, sdk)
    sdk.on_request(rh.handle_request)
    while True:
        await asyncio.sleep(1)


asyncio.run(main())
