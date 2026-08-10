print("Starting test.py")

import os

# Get current directory
current_dir = os.getcwd()+"/test-files"

# List all files and directories
items = sorted(os.listdir(current_dir))

print("Contents of", current_dir)
for item in items:
    print(item)

x = ""
with open(current_dir+"/input.csv", "r", encoding="utf-8") as f:
    f.readline()
    x = f.readline()

print("Writing output...")

with open("py-output.txt", "w", encoding="utf-8") as f:
    f.write("Example output from python\n")
    f.write(x)
    f.write("The end.")

print("Finishing test.py")
