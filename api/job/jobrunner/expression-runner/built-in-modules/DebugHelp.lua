DebugHelp = {}

function DebugHelp.listAllTables(offset, story, recursive)
    local n,v
    for n,v in pairs(story) do
        if n ~= "loaded" and n ~= "_G" then
            io.write (offset .. n .. " " )
            print (v)
            if type(v) == "table" and recursive then
                DebugHelp.listAllTables(offset .. "--> ",v)
            end
        end
    end
end

function DebugHelp.printTable(name, t)
    local d = DebugHelp.dump(t, "", 30)
    print(name..": " .. d .. "\n")
end

function DebugHelp.dump(o, prevPrefix, keyLimit)
    local prefix = prevPrefix.."  "

    if type(o) == 'table' then
        local s = prevPrefix..'{\n'
        local c = 0
        for k,v in pairs(o) do
                if type(k) ~= 'number' then
                    k = '"'..k..'"'
                end
            s = s .. prefix .. '['..k..'] = ' .. DebugHelp.dump(v, prefix, 10) .. '\n'

                if c > keyLimit then
                    s = s .. prefix .. "... (".. (#o-c) .. " more, " .. #o .. " total items)...\n"
                    break
                end

            c = c + 1
        end

        return s .. prevPrefix .. '}\n'
    else
        return tostring(o)
    end
end

lastUnix = 0
function DebugHelp.logPerf(str)
    local now = os.clock()
    print(str.."   elapsed: "..string.format("%.4f", now-lastUnix)..", clock: "..string.format("%.4f", now))
    lastUnix = now
end

return DebugHelp