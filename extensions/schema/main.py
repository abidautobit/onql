import asyncio
import os

from onql_ext_sdk import SDK

import request_handler
import schema


async def main():
    sdk = await SDK.create("schema", nats_url="nats://nats:4222")
    sc = schema.Schema(sdk)

    rh = request_handler.RequestHandler(sc, sdk)
    sdk.on_request(rh.handle_request)
    while True:
        await asyncio.sleep(1)


asyncio.run(main())
