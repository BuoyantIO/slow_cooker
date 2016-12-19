slow_cooker = require('slow_cooker')
json = require('json')

local Built = {}

local AddLookup = function(charset)
  local chars = {}
  for i = 0, 255 do
    chars[i + 1] = string.char(i)
  end
  local sub = string.gsub(table.concat(chars), '[^' .. charset .. ']', '')
  local lookup = {}
  for i = 1, string.len(sub) do
    lookup[i] = string.sub(sub, i, i)
  end
  Built[charset] = lookup

  return lookup
end

function string.random(len, charset)
  -- len (number)
  -- charset (string, optional); e.g. %l%d for lower case letters and digits

  local charset = charset or '%w'

  if charset == '' then
    return ''
  else
    local res = {}
    local lookup = Built[charset] or AddLookup(charset)
    local range = #lookup

    for i = 1, len do
      res[i] = lookup[math.random(1, range)]
    end

    return table.concat(res)
  end
end

function slow_cooker.generate_data(method, url, host, reqID)
  local body = {method=method, url=url, host=host, request_id=reqID, random_string=string.random(50)}
  return json.encode(body)
end
