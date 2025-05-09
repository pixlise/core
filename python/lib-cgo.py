
from pixlise import Pixlise

pixlib = Pixlise()
pixlib.authenticate()

#result = pixlib.testStrings("scanId123", 456)
#print("python result: %s" % result)

#result = pixlib.testIntArray("scanId789", 111)
#print("python result: %s" % result)

# print("python result len:", result, type(result), type(result[0]), sys.getsizeof(result[0]), "\n")
# print(result[0], result[1], result[2])

# spectrum = pixlib.getSpectrum("500302337", 15, 3, "A")
# print(spectrum)
# print(len(spectrum))

scans = pixlib.listScans("") #"500302337")
print(scans)
print(len(scans.scans))

# labels = pixlib.getScanMetaList("500302337")
# print(labels)

# meta = pixlib.getScanMetaData("500302337")
# print(meta)

# cols = pixlib.getScanEntryDataColumns("500302337")
# print(cols)

# housekeeping = pixlib.getScanEntryDataColumn("500302337", "f_head_yellow_piece")
# print(housekeeping)

# spectra = pixlib.getScanSpectra("500302337")
# print(spectra)

# quants = pixlib.listScanQuants("500302337")
# print(quants)

# quant = pixlib.getQuant("quant-umwzkcv6cmy06e36", False)
# print(quant)

# cols = pixlib.getQuantColumns("quant-umwzkcv6cmy06e36")
# print(cols)

# quant = pixlib.getQuant("quant-umwzkcv6cmy06e36", True)
# print(quant)

# quantMap = pixlib.getQuantColumn("quant-umwzkcv6cmy06e36", "Na2O_%", "Combined")
# print(quantMap)

# quantMap = pixlib.getQuantColumn("quant-umwzkcv6cmy06e36", "chisq", "Combined")
# print(quantMap)

# quantMap = pixlib.getQuantColumn("quant-umwzkcv6cmy06e36", "total_counts", "Combined")
# print(quantMap)

# images = pixlib.listScanImages(["500302337"], False)
# print(images)

# rois = pixlib.listScanROIs("500302337")
# print(rois)

# roi = pixlib.getROI("297n3r25d0870t8b", False)
# print(roi)

# xyzs = pixlib.getScanBeamLocations("500302337")
# print(xyzs)

# entries = pixlib.getScanEntries("500302337")
# print(entries)

# vers = pixlib.getScanImageBeamLocationVersions("500302337/PCW_1377_0789190464_000RCM_N064227850030233700030LUJ01.png")
# print(vers)

# ijs = pixlib.getScanImageBeamLocations("500302337/PCW_1377_0789190464_000RCM_N064227850030233700030LUJ01.png", "500302337", 3)
# print(ijs)

# ijs = pixlib.getScanImageBeamLocations("500302337/PCW_1377_0789190464_000RCM_N064227850030233700030LUJ01.png", "500302337", -1)
# print(ijs)

# spectra = pixlib.getScanSpectra("500302337")
# print(spectra)

# diffraction = pixlib.getDetectedDiffractionPeaks("101384711")
# print(diffraction)

# roi = pixlib.allocROI([7, 8, 10, 11])
# roi.name = "peters ROI"
# roi.scanId = "500302337"
# roi.description = "Lib created ROI"
# #roi.scanEntryIndexesEncoded = 

# roiResp = pixlib.createROI(roi, False)
# print(roiResp)
