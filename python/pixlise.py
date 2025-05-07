import array as arr
from ctypes import *

from pixlisemsgs import scan_pb2
from pixlisemsgs import scan_msgs_pb2
from pixlisemsgs import scan_entry_metadata_msgs_pb2
from pixlisemsgs import quantification_retrieval_msgs_pb2
from pixlisemsgs import image_msgs_pb2
from pixlisemsgs import roi_msgs_pb2
from pixlisemsgs import scan_beam_location_msgs_pb2
from pixlisemsgs import scan_entry_msgs_pb2
from pixlisemsgs import image_beam_location_msgs_pb2
from pixlisemsgs import spectrum_msgs_pb2
from pixlisemsgs import diffraction_detected_peak_msgs_pb2
from pixlisemsgs import roi_pb2

from google.protobuf.json_format import MessageToJson


# import os
# protopath = os.getcwd() + "/pixlisemsgs.zip"

# from zipimport import zipimporter
# importer = zipimporter(protopath)
# importer.get_filename("scan_msgs_pb2")
# scan_msgs_pb2 = importer.load_module("scan_msgs_pb2")


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


def _popArray():
    global _arrays
    arr = _arrays[0]
    _arrays = _arrays[1:]
    return arr

#####################################################
# Error struct to use Go returned errors
#####################################################
class Error(Structure):
    _fields_ = [('err', c_char_p)]

    # NOTE: This thing will leak memory... Look at cleanup code in: https://fluhus.github.io/snopher/

#####################################################
# Helpers to reduce code duplication
#####################################################

def readArrayResult(msgName, err, parseInto):
    if len(err) > 0:
        print(msgName, "error:", err)
        return None
    
    parseInto.ParseFromString(bytes(_popArray()))
    return parseInto

#####################################################
    

class Pixlise:
    def __init__(self) -> None:
        self._lib = CDLL(PIXLISE_SHARED_LIBRARY_PATH)
        self._lib.authenticate.argtypes = [alloc_f]
        self._lib.authenticate.restype = c_char_p
        self._lib.listScans.argtypes = [Structure]
        self._lib.listScans.restype = c_char_p
        self._lib.getScanMetaList.argtypes = [Structure]
        self._lib.getScanMetaList.restype = c_char_p
        self._lib.getScanMetaData.argtypes = [Structure]
        self._lib.getScanMetaData.restype = c_char_p
        self._lib.getScanEntryDataColumns.argtypes = [Structure]
        self._lib.getScanEntryDataColumns.restype = c_char_p
        self._lib.getScanEntryDataColumn.argtypes = [Structure, Structure]
        self._lib.getScanEntryDataColumn.restype = c_char_p
        self._lib.getScanSpectra.argtypes = [Structure]
        self._lib.getScanSpectra.restype = c_char_p
        self._lib.listScanQuants.argtypes = [Structure]
        self._lib.listScanQuants.restype = c_char_p
        self._lib.getQuant.argtypes = [Structure, c_bool]
        self._lib.getQuant.restype = c_char_p
        self._lib.getQuantColumns.argtypes = [Structure]
        self._lib.getQuantColumns.restype = c_char_p
        self._lib.getQuantColumn.argtypes = [Structure, Structure, Structure]
        self._lib.getQuantColumn.restype = c_char_p
        self._lib.listScanImages.argtypes = [Structure, c_bool]
        self._lib.listScanImages.restype = c_char_p
        self._lib.listScanROIs.argtypes = [Structure]
        self._lib.listScanROIs.restype = c_char_p
        self._lib.getROI.argtypes = [Structure, c_bool]
        self._lib.getROI.restype = c_char_p
        self._lib.getScanBeamLocations.argtypes = [Structure]
        self._lib.getScanBeamLocations.restype = c_char_p
        self._lib.getScanEntries.argtypes = [Structure]
        self._lib.getScanEntries.restype = c_char_p
        self._lib.getScanImageBeamLocationVersions.argtypes = [Structure]
        self._lib.getScanImageBeamLocationVersions.restype = c_char_p
        self._lib.getScanImageBeamLocations.argtypes = [Structure, Structure, c_int32]
        self._lib.getScanImageBeamLocations.restype = c_char_p
        self._lib.getDetectedDiffractionPeaks.argtypes = [Structure]
        self._lib.getDetectedDiffractionPeaks.restype = c_char_p
        self._lib.createROI.argtypes = [Structure, c_bool]
        self._lib.createROI.restype = c_char_p
        

        # del_error = self._lib.delError
        # del_error.argtypes = [Error]

    def authenticate(self):
        return self._lib.authenticate(my_alloc)

    # def getSpectrum(self, scanId: str, pmc: int, spectrumType: int, detector: str):
    #     err = self._lib.getSpectrum(my_alloc, makeGoString(scanId), pmc, spectrumType, makeGoString(detector))
    #     print("getSpectrum err", err)
    #     if len(err) > 0:
    #         return []
    
    #     return _popArray()

    # def testStrings(self, scanId: str, pmc: int) -> str:
    #     r = self._lib.testStrings(makeGoString(scanId), pmc)
    #     # NOTE: r leaks memory! Need C.free ???
    #     return r

    # def testIntArray(self, scanId: str, pmc: int) -> Array:
    #     self._lib.testIntArray(my_alloc, makeGoString(scanId), pmc)
    #     return _popArray()


    def listScans(self, scanId: str):
        return readArrayResult("listScans", self._lib.listScans(makeGoString(scanId)), scan_msgs_pb2.ScanListResp())

    def getScanMetaList(self, scanId: str):
        return readArrayResult("getScanMetaList", self._lib.getScanMetaList(makeGoString(scanId)), scan_msgs_pb2.ScanMetaLabelsAndTypesResp())

    def getScanMetaData(self, scanId: str):
        return readArrayResult("getScanMetaData", self._lib.getScanMetaData(makeGoString(scanId)), scan_entry_metadata_msgs_pb2.ScanEntryMetadataResp())

    def getScanEntryDataColumns(self, scanId: str):
        return readArrayResult("getScanEntryDataColumns", self._lib.getScanEntryDataColumns(makeGoString(scanId)), scan_pb2.ClientStringList())

    def getScanEntryDataColumn(self, scanId: str, columnName: str):
        return readArrayResult("getScanEntryDataColumn", self._lib.getScanEntryDataColumn(makeGoString(scanId), makeGoString(columnName)), scan_pb2.ClientMap())

    def getScanSpectra(self, scanId: str):
        return readArrayResult("getScanSpectra", self._lib.getScanSpectra(makeGoString(scanId)), spectrum_msgs_pb2.SpectrumResp())

    def listScanQuants(self, scanId: str):
        return readArrayResult("listScanQuants", self._lib.listScanQuants(makeGoString(scanId)), quantification_retrieval_msgs_pb2.QuantListResp())

    def getQuant(self, quantId: str, summaryOnly: bool):
        return readArrayResult("getQuant", self._lib.getQuant(makeGoString(quantId), summaryOnly), quantification_retrieval_msgs_pb2.QuantGetResp())

    def listScanImages(self, scanIds: list[str], mustIncludeAll: bool):
        if not (scanIds and isinstance(scanIds, list) and all(isinstance(scanId, str) for scanId in scanIds)):
            raise TypeError('scanIds must be a list of one or more string ids')

        goScanIds = "|".join(scanIds)

        return readArrayResult("listScanImages", self._lib.listScanImages(makeGoString(goScanIds), mustIncludeAll), image_msgs_pb2.ImageListResp())

    def listScanROIs(self, scanId: str):
        return readArrayResult("listScanROIs", self._lib.listScanROIs(makeGoString(scanId)), roi_msgs_pb2.RegionOfInterestListResp())

    def getROI(self, id: str, isMist: bool):
        return readArrayResult("getROI", self._lib.getROI(makeGoString(id), isMist), roi_msgs_pb2.RegionOfInterestGetResp())

    def getScanBeamLocations(self, scanId: str):
        return readArrayResult("getScanBeamLocations", self._lib.getScanBeamLocations(makeGoString(scanId)), scan_beam_location_msgs_pb2.ScanBeamLocationsResp())

    def getScanEntries(self, scanId: str):
        return readArrayResult("getScanEntries", self._lib.getScanEntries(makeGoString(scanId)), scan_entry_msgs_pb2.ScanEntryResp())

    def getScanImageBeamLocationVersions(self, imageName: str):
        return readArrayResult("getScanImageBeamLocationVersions", self._lib.getScanImageBeamLocationVersions(makeGoString(imageName)), image_beam_location_msgs_pb2.ImageBeamLocationVersionsResp())

    def getScanImageBeamLocations(self, imageName: str, scanId: str, version: int):
        return readArrayResult("getScanImageBeamLocations", self._lib.getScanImageBeamLocations(makeGoString(imageName), makeGoString(scanId), version), image_beam_location_msgs_pb2.ImageBeamLocationsResp())

    def getDetectedDiffractionPeaks(self, scanId: str):
        return readArrayResult("getDetectedDiffractionPeaks", self._lib.getDetectedDiffractionPeaks(makeGoString(scanId)), diffraction_detected_peak_msgs_pb2.DetectedDiffractionPeaksResp())

    def getQuantColumns(self, quantId: str):
        return readArrayResult("getQuantColumns", self._lib.getQuantColumns(makeGoString(quantId)), scan_pb2.ClientStringList())

    def getQuantColumn(self, quantId: str, columnName: str, detector: str):
        return readArrayResult("getQuantColumn", self._lib.getQuantColumn(makeGoString(quantId), makeGoString(columnName), makeGoString(detector)), scan_pb2.ClientMap())
   
    def allocROI(self, pmcs: list[int]) -> roi_pb2.ROIItem:
        item = roi_pb2.ROIItem()
        for pmc in pmcs:
            item.scanEntryIndexesEncoded.append(pmc)
        return item

    def createROI(self, roi: roi_pb2.ROIItem, isMist: bool):
        # Encode item to protobuf byte array
        roiJSON = MessageToJson(roi)
        return readArrayResult("createROI", self._lib.createROI(makeGoString(roiJSON), isMist), roi_msgs_pb2.RegionOfInterestWriteResp())
