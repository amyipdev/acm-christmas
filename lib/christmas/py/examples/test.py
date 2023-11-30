import acm_christmas
import asyncio


async def main():
    x = acm_christmas.TreeConnection(token="", dest="localhost")
    await x.connect()
    
    
loop = asyncio.get_event_loop()
loop.run_until_complete(main())
