print("Starting test.lua")

local lfs = require("lfs")

-- Get current directory
local current_dir = lfs.currentdir() .. "/test-files"

print("Contents of " .. current_dir)

-- List all files and directories
for item in lfs.dir(current_dir) do
    if item ~= "." and item ~= ".." then
        print(item)
    end
end

-- Read from input.csv
local x = ""
local input_path = current_dir .. "/input.csv"
local file = io.open(input_path, "r")

if file then
    file:read("*l")       -- skip first line
    x = file:read("*l") or ""
    file:close()
else
    error("Could not open file: " .. input_path)
end

print("Writing output...")

-- Write to lua-output.txt
local output = io.open("lua-output.txt", "w")

if output then
    output:write("Example output from lua\n")
    output:write(x .. "\n")
    output:write("The end.")
    output:close()
else
    error("Could not open output file")
end

print("Finishing test.lua")