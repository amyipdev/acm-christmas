# SPDX-License-Identifier: GPL-3.0-or-later

import websockets
import numpy as np

# make sure to regenerate using genpb.sh before build
from . import christmas_pb2 as cp


class TreeConnection:
    # one TreeConnection per credentials
    # if they change, make a new TreeConnection
    def __init__(self, token: str, dest: str):
        self.token = token
        self.dest = dest
        self.ws = None
        self.connected = False
        self.ix = 0  # image width
        self.iy = 0  # image height
        self.lc = 0  # led count
        
    # TODO: note that if connection fails, perfectly
    # legitimate to .connect() again
    async def connect(self):    
        self.ws = await websockets.connect(f"ws://{self.dest}/ws")
        auth = cp.AuthenticateRequest()
        auth.secret = self.token
        if not auth.IsInitialized():
            raise ValueError("Failed to create AuthenticateRequest")
        resp = self._send(auth)
        if not resp.message.success:
            raise PermissionError("Invalid token")
        self.connected = True
        resp = await self._send(cp.GetLEDCanvasInfoRequest())
        self.ix = resp.message.width
        self.iy = resp.message.height
        # TODO: use GetLEDs once that endpoint is made
        resp = await self._send(cp.GetLEDsRequest())
        self.lc = len(resp.message.leds)
        
    async def send_image(self, img: np.array):
        self._check_connected()
        assert img.dtype == np.uint8
        assert img.size == self.ix * self.iy * 4
        msg = cp.SetLEDCanvasRequest()
        msg.pixels = img
        await self._send_lt(msg)

    async def send_raw_pixels(self, pxs: np.array):
        self._check_connected()
        assert pxs.dtype == np.uint64
        assert pxs.size == self.lc
        msg = cp.SetLEDsRequest()
        msg.leds = pxs
        await self._send_lt(msg)

    async def close(self):
        # because .close() is idempotent, no need to _check_connected()
        await self.ws.close()
        self.connected = False
        
    async def _send(self, msg) -> cp.LEDServerMessage:
        await self._send_lt(msg)
        resp = cp.LEDServerMessage.ParseFromString(await self.ws.recv())
        _check_error(resp)
        return resp

    async def _send_lt(self, msg):
        await self.ws.send(_form_msg(msg).SerializeToString)

    def _check_connected(self):
        if not self.connected:
            raise ConnectionError("No connection established, must run .connect()")
        
        
def _check_error(resp):    
    if hasattr(resp, "error"):
        raise Exception(resp.error)


def _form_msg(msg):
    cmsg = cp.LEDClientMessage()
    cmsg.message = msg
    return cmsg
