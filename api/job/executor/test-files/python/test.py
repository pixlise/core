print("Starting test.py")

import os
import numpy as np

# Get current directory
current_dir = os.getcwd()+"/Input"

# List all files and directories
items = os.listdir(current_dir)

print("Contents of", current_dir)
for item in items:
    print(item)

# x = ""
# with open(current_dir+"/input.csv", "r", encoding="utf-8") as f:
#     f.readline()
#     x = f.readline()

arr = np.loadtxt(current_dir+"/input.csv", delimiter=",", dtype=str)
x = arr[2,1]

print("Writing output...")

with open("py-output.txt", "w", encoding="utf-8") as f:
    f.write("Example output from python\n")
    f.write(arr[2,0])
    f.write(", ")
    f.write(arr[2,1])
    f.write("\nThe end.")

print("Finishing test.py")
