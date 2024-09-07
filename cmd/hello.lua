local fmt = require("fmt")
local email = require("email")

-- Configuration
local config = {
    username = os.getenv("EMAIL_USERNAME"),
    password = os.getenv("EMAIL_PASSWORD"),
    imap_server = "imap.mail.me.com",
    imap_port = "993",
    search_keyword = "yt-dlp] Release",
    -- search_keyword = "Release",
    -- bash_script_path = "/path/to/your/script.sh",
    -- slack_webhook_url = "https://hooks.slack.com/services/PATH/TO/WEBHOOK"
}

-- IMAP client functions
local function connect_imap()
    print("Connecting to IMAP server...")

    local client, err  = email.new(config.imap_server, config.imap_port)
    if err then
        fmt.Println("Error connecting to IMAP server: %s", err)
        return
    end

    return client
end

-- Main loop
while true do
    local client = connect_imap()
    if not client then
        print("Error connecting to IMAP server")
        os.exit(1)
    end

    local err = client.Login(client, config.username, config.password)
    if err then
        fmt.Printf("Error logging in to IMAP server: %v\n", err)
        return
    else
        print("Logged in successfully")
    end

    err = client.SelectMailbox(client, "INBOX")
    if err then
        fmt.Printf("Error selecting mailbox: %v\n", err)
        return
    end

    local seqNums
    seqNums, err = client:Search(config.search_keyword)
    if err then
        fmt.Printf("Error searching mailbox: %v\n", err)
        return
    end

    fmt.Printf("seqNums: %v\n", seqNums)

    local array = email.getArray(seqNums)
    -- if #seqNums == 0 then
    if #array == 0 then
        print("No matching emails found")
    else
        print("Found matching email(s):")
        for _, seqNum in pairs(array) do
            print("  - " .. seqNum)
        end
    end

    err = client.Logout(client)
    if err then
        fmt.Printf("Error logging out: %v\n", err)
        return
    else
        print("Logged out successfully")
    end


    os.execute("sleep " .. tonumber(60))
end
