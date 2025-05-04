local ok = redis.call("set", KEYS[1], ARGV[1], "NX", "PX", ARGV[2])
if ok then
    return 0 -- lock set successfully
else
    local val = redis.call("get", KEYS[1])
    if val == ARGV[1] then
        return 1 -- already holding the lock 
    else
        return -1 -- lock is held by another
    end
end
