import socket
import os, os.path
import time
from collections import deque

if os.path.exists("/tmp/TelldusEvents"):
    os.remove("/tmp/TelldusEvents")

command_1 = "16:TDRawDeviceEvent93:class:command;protocol:arctech;model:selflearning;house:902538;unit:4;group:0;method:turnoff;i1s\n"

server = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
server.bind("/tmp/TelldusEvents")
while True:
    server.listen(1)
    conn, addr = server.accept()
    while True:
        conn.sendall(bytes(command_1, 'utf-8'))
        time.sleep(5)
    # datagram = conn.recv(1024)
    # if datagram:
    #     tokens = datagram.strip().split()
    # conn.close()
