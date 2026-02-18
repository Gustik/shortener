counter = 0

wrk.method = "POST"
wrk.headers["Content-Type"] = "text/plain"

request = function()
    counter = counter + 1
    wrk.body = "https://example.com/" .. counter .. "/" .. math.random(1, 1000000)
    return wrk.format()
end
