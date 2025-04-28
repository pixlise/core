import sys
from pixlise import Pixlise

pixlib = Pixlise()

configPath = "/mnt/c/Users/peter/Documents/code/PIXLISE/core/core/client/internal/config.json"
pixlib.authenticate(configPath)

#result = pixlib.testStrings("scanId123", 456)
#print("python result: %s" % result)

#result = pixlib.testIntArray("scanId789", 111)
#print("python result: %s" % result)

# print("python result len:", result, type(result), type(result[0]), sys.getsizeof(result[0]), "\n")
# print(result[0], result[1], result[2])

spectrum = pixlib.getSpectrum("500302337", 15, 3, "A")
print(spectrum)
print(len(spectrum))
