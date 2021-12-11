local task_queue_key = KEYS[1]
local worker_task_key = KEYS[2]
local now_str = ARGV[1]
local OK = 'OK'

local in_progress_task = redis.call('GET', worker_task_key)
if in_progress_task then
    return in_progress_task
end

local tasks = redis.call('ZRANGEBYSCORE', task_queue_key, '0', now_str, 'LIMIT', 0, 1)
local task = tasks[1]
if not task then
    return OK
end

redis.call('SET', worker_task_key, task)
redis.call('ZREM', task_queue_key, task)

return task