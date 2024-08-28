-- local socket = require("socket")
local fmt = import("fmt")
local socket = import("socket")
-- local os = import("os")
-- local ssl = require("ssl")
-- local http = require("socket.http")
-- local ltn12 = require("ltn12")
-- local json = require("dkjson")
-- local lume = require("lume")

-- Configuration
local config = {
    username = "",
    password = "",
    imap_server = "imap.mail.me.com",
    imap_port = 993,
    search_keyword = "Release",
    -- bash_script_path = "/path/to/your/script.sh",
    -- slack_webhook_url = "https://hooks.slack.com/services/PATH/TO/WEBHOOK"
}

-- IMAP client functions
local function connect_imap()
    print("Connecting to IMAP server...")

    -- local sock = assert(socket.tcp())
    -- sock:settimeout(10)
    -- assert(sock:connect(config.imap_server, config.imap_port))

    -- local params = {
    --     mode = "client",
    --     protocol = "tlsv1_2",
    --     verify = "none",
    --     options = "all"
    -- }

    -- local conn = assert(ssl.wrap(sock, params))
    -- assert(conn:dohandshake())

    -- return conn
end

local function send_command(conn, command)
    conn:send(command .. "\r\n")
    local response = {}
    while true do
        local line = conn:receive("*l")
        table.insert(response, line)
        if line:match("^%* OK") or line:match("^%d+ OK") or line:match("^%d+ BAD") or line:match("^%d+ NO") then
            break
        end
    end
    return response
end

local function login(conn, username, password)
    send_command(conn, string.format('A1 LOGIN "%s" "%s"', username, password))
end

local function search_emails(conn, criteria)
    local response = send_command(conn, 'A2 SEARCH ' .. criteria)
    local ids = {}
    for _, line in ipairs(response) do
        for id in line:gmatch("%d+") do
            table.insert(ids, id)
        end
    end
    return ids
end

local function fetch_subject(conn, id)
    local response = send_command(conn, string.format('A3 FETCH %s (BODY[HEADER.FIELDS (SUBJECT)])', id))
    for _, line in ipairs(response) do
        local subject = line:match("Subject: (.+)")
        if subject then
            return subject
        end
    end
    return nil
end

-- Slack message function
local function send_slack_message(message)
    local payload = json.encode({text = message})
    local request_body = payload
    local response_body = {}

    local res, code, response_headers = http.request {
        url = config.slack_webhook_url,
        method = "POST",
        headers = {
            ["Content-Type"] = "application/json",
            ["Content-Length"] = #request_body
        },
        source = ltn12.source.string(request_body),
        sink = ltn12.sink.table(response_body)
    }

    if code ~= 200 then
        print("Error sending Slack message: " .. code)
    else
        print("Slack message sent successfully")
    end
end

-- Main loop
while true do
    local conn = connect_imap()
    os.execute("sleep " .. tonumber(2))
    -- login(conn, config.username, config.password)
    -- send_command(conn, 'A4 SELECT INBOX')

    -- local email_ids = search_emails(conn, 'UNSEEN')

    -- for _, id in ipairs(email_ids) do
    --     local subject = fetch_subject(conn, id)
    --     if subject and subject:lower():find(config.search_keyword) then
    --         print("Found matching email: " .. subject)

    --     end
    -- end

    -- conn:close()
    -- socket.sleep(60)  -- Wait for 60 seconds before checking again
end
