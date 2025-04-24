#import subprocess
import os


def authenticate():
    #subprocess.run(["ls", "-l"]) 
    #os.system("go run ../core/client/client.go")
    os.system("go run core\\client\\internal\\cmdline\\main.go --configFile core\\client\\internal\\config.json")

#def getSpectra(scanId, pmc, detector)
    

authenticate()
