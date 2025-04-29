local val = redis.call("get", KEYS[1])
if not val then
    return -1 -- the key does not exist
elseif val == ARGV[1] then
    redis.call("expire", KEYS[1], ARGV[2])
    return 0 -- successfully extended the lock
else
    return 1  -- lock is held by another node
end
