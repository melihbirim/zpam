-- @name lua-domain-checker
-- @version 1.0.0
-- @description ZPAM plugin for reputation-checker
-- @type reputation-checker
-- @interfaces   - "ReputationChecker"

-- ZPAM reputation-checker Plugin
-- TODO: Implement your spam detection logic here

-- Main function that ZPAM will call
-- @param email table - Email data with fields: from, to, subject, body, headers, attachments
-- @return table - Result with score, confidence, rules, metadata
function check_reputation(email)
    -- TODO: Implement your reputation-checker logic here
    
    local result = {
        score = 0.0,       -- 0.0 to 100.0 (higher = more spam)
        confidence = 0.7,  -- 0.0 to 1.0 (confidence in the score)
        rules = {},        -- Array of triggered rule descriptions
        metadata = {}      -- Key-value pairs of additional information
    }
    
    -- Example analysis based on plugin type
        -- Example: Check sender domain reputation
    local domain = extract_domain(email.from)
    local suspicious_domains = {"tempmail.com", "guerrillamail.com", "10minutemail.com"}
    
    for _, bad_domain in ipairs(suspicious_domains) do
        if domain == bad_domain then
            result.score = 90.0
            result.confidence = 0.95
            table.insert(result.rules, "Known spam domain: " .. domain)
            break
        end
    end
    
    -- Check for suspicious TLDs
    if string.match(domain, "%.tk$") or string.match(domain, "%.ml$") then
        result.score = result.score + 30.0
        table.insert(result.rules, "Suspicious TLD")
    end
    
    -- Add metadata
    result.metadata.plugin_name = "lua-domain-checker"
    result.metadata.version = "1.0.0"
    result.metadata.analysis_type = "reputation-checker"
    
    return result
end

-- Helper functions for common tasks
function contains_keyword(text, keywords)
    if not text or not keywords then
        return false
    end
    
    local lower_text = string.lower(text)
    for _, keyword in ipairs(keywords) do
        if string.find(lower_text, string.lower(keyword), 1, true) then
            return true
        end
    end
    return false
end

function count_caps(text)
    if not text then return 0 end
    local caps = 0
    for i = 1, #text do
        local char = string.sub(text, i, i)
        if char:match("%u") then
            caps = caps + 1
        end
    end
    return caps / #text
end

function extract_domain(email_addr)
    if not email_addr then return "" end
    local at_pos = string.find(email_addr, "@")
    if at_pos then
        return string.sub(email_addr, at_pos + 1)
    end
    return email_addr
end

-- ZPAM API functions available:
-- zpam.log(message)              - Log a message
-- zpam.contains(text, pattern)   - Case-insensitive substring search
-- zpam.domain_from_email(email)  - Extract domain from email address
