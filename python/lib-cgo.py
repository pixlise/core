#import os
#print(os.getcwd())

#C:\\Users\\peter\\Documents\\code\\PIXLISE\\core\\client\\lib\\pixlise.so
#C:\\Users\\peter\\Documents\\code\\PIXLISE\\core
#C:\\Users\\peter\\Documents\\code\\PIXLISE\\core\\core\\client\\lib\\pixlise.so

from ctypes import *
from ctypes import cdll

#pixlib = ctypes.cdll.LoadLibrary('C:\\Users\\peter\\Documents\\code\\PIXLISE\\core\\core\\client\\lib\\pixlise.so')
#pixlib = ctypes.cdll.LoadLibrary('./client/lib/pixlise.so')
#pixlib = ctypes.cdll.LoadLibrary('C:\\Users\\peter\\Documents\\code\\PIXLISE\\core\\client\\lib\\pixlise.so')
pixlib = cdll.LoadLibrary('../core/client/lib/pixlise-linux.so')
getSpectrum = pixlib.getSpectrum

def authenticate(cfgPath):
    pixlib.authenticate(makeGoString(cfgPath))

class go_string(Structure):
    _fields_ = [
        ("p", c_char_p),
        ("n", c_int)]


def makeGoString(str):
    return go_string(c_char_p(str.encode('utf-8')), len(str))

#strr = "hello"
#configPath = go_string(c_char_p(strr), len(strr))

# windows path:
#configPath = "C:\\Users\\peter\\Documents\\code\\PIXLISE\\core\\core\\client\\internal\\config.json"
# wsl path:
configPath = "/mnt/c/Users/peter/Documents/code/PIXLISE/core/core/client/internal/config.json"

#authenticate(configPath)

spectra = getSpectrum(makeGoString("500302337"), 15, 3, makeGoString("A"))
print(len(spectra.r0))