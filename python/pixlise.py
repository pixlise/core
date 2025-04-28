import array as arr
from ctypes import *
from dataclasses import dataclass


PIXLISE_SHARED_LIBRARY_PATH = "../core/client/lib/pixlise-linux.so"


class go_SpectrumResult(Structure):
    _fields_ = [
        ("r0", c_char_p),
        ("r1", c_int)]

class go_Int32Array(Structure):
    _fields_ = [
        ("data", c_char_p),
        ("len", c_int),
        ("cap", c_int)]
    
#####################################################
# Go string conversion
#####################################################

class go_string(Structure):
    _fields_ = [
        ("p", c_char_p),
        ("n", c_int)]

def makeGoString(str):
    return go_string(c_char_p(str.encode('utf-8')), len(str))

#####################################################
# Needed to be able to allocate memory from Go
#####################################################

# A function that receives an array type string and a size,
# and returns a pointer.
alloc_f = CFUNCTYPE(c_void_p, c_char_p, c_int64)

_arrays: list[arr.array] = []

@alloc_f
def my_alloc(typecode, size):
    #print("my_alloc", typecode, size)
    allocdArray = arr.array(typecode.decode(), (0 for _ in range(size)))
    #print(allocdArray)
    _arrays.append(allocdArray)
    return allocdArray.buffer_info()[0]


#####################################################
# Error struct to use Go returned errors
#####################################################
class Error(Structure):
    _fields_ = [('err', c_char_p)]

    # NOTE: This thing will leak memory... Look at cleanup code in: https://fluhus.github.io/snopher/

#####################################################


class Pixlise:
    def __init__(self) -> None:
        self._lib = CDLL(PIXLISE_SHARED_LIBRARY_PATH)
        self._lib.authenticate.argtypes = [Structure]
        self._lib.getSpectrum.argtypes = [alloc_f, Structure, c_int, c_int, Structure]
        self._lib.getSpectrum.restype = c_char_p
        self._lib.testStrings.argtypes = [Structure, c_int]
        self._lib.testStrings.restype = c_char_p
        self._lib.testIntArray.argtypes = [alloc_f, Structure, c_int]

        # del_error = self._lib.delError
        # del_error.argtypes = [Error]

    def authenticate(self, cfgPath: str):
        return self._lib.authenticate(makeGoString(cfgPath))

    def getSpectrum(self, scanId: str, pmc: int, spectrumType: int, detector: str):
        err = self._lib.getSpectrum(my_alloc, makeGoString(scanId), pmc, spectrumType, makeGoString(detector))
        print("getSpectrum err", err)
        if len(err) > 0:
            return []
    
        return _arrays[0]

    def testStrings(self, scanId: str, pmc: int) -> str:
        r = self._lib.testStrings(makeGoString(scanId), pmc)
        # NOTE: r leaks memory! Need C.free ???
        return r

    def testIntArray(self, scanId: str, pmc: int) -> Array:
        self._lib.testIntArray(my_alloc, makeGoString(scanId), pmc)
        return _arrays[0]
