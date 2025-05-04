local val = redis.call("get", KEYS[1])
if not val then
    return -1 -- key does not exist
elseif val == ARGV[2] then
    redis.call("pexpire", KEYS[1], ARGV[1])
    return 0 -- successfully extended
else
    return 1 -- lock is held by another node
end
