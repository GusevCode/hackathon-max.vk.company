box.cfg {
    listen = 3301,
    memtx_memory = 256 * 1024 * 1024,
    wal_mode = "write",
}

box.once("application_schema_v2", function()
    local user = os.getenv("TARANTOOL_USER") or "app"
    local password = os.getenv("TARANTOOL_PASSWORD") or "change-me"

    if not box.schema.user.exists(user) then
        box.schema.user.create(user, { password = password })
    end
    box.schema.user.grant(user, "read,write,execute", "universe")

    local organizations = box.schema.space.create("organizations", { if_not_exists = true })
    organizations:format({
        { name = "id", type = "string" },
        { name = "name", type = "string" },
        { name = "created_at", type = "number" },
        { name = "is_active", type = "boolean" },
    })
    organizations:create_index("primary", { parts = { "id" }, if_not_exists = true })

    local users = box.schema.space.create("users", { if_not_exists = true })
    users:format({
        { name = "id", type = "string" },
        { name = "organization_id", type = "string" },
        { name = "max_user_id", type = "unsigned" },
        { name = "display_name", type = "string" },
        { name = "roles", type = "array" },
        { name = "status", type = "string" },
        { name = "created_at", type = "number" },
        { name = "manager_id", type = "string", is_nullable = true },
    })
    users:create_index("primary", { parts = { "id" }, if_not_exists = true })
    users:create_index("max_user_id", { parts = { "max_user_id" }, unique = false, if_not_exists = true })
    users:create_index("organization_id", { parts = { "organization_id" }, unique = false, if_not_exists = true })

    local objects = box.schema.space.create("objects", { if_not_exists = true })
    objects:format({
        { name = "id", type = "string" },
        { name = "organization_id", type = "string" },
        { name = "name", type = "string" },
        { name = "address", type = "string" },
        { name = "kind", type = "string" },
        { name = "created_at", type = "number" },
    })
    objects:create_index("primary", { parts = { "id" }, if_not_exists = true })
    objects:create_index("organization_id", { parts = { "organization_id" }, unique = false, if_not_exists = true })

    local work_types = box.schema.space.create("work_types", { if_not_exists = true })
    work_types:format({
        { name = "id", type = "string" },
        { name = "organization_id", type = "string" },
        { name = "name", type = "string" },
        { name = "created_at", type = "number" },
    })
    work_types:create_index("primary", { parts = { "id" }, if_not_exists = true })
    work_types:create_index("organization_id", { parts = { "organization_id" }, unique = false, if_not_exists = true })

    local invites = box.schema.space.create("invites", { if_not_exists = true })
    invites:format({
        { name = "code", type = "string" },
        { name = "organization_id", type = "string" },
        { name = "roles", type = "array" },
        { name = "manager_id", type = "string" },
        { name = "expires_at", type = "number" },
        { name = "used_by", type = "unsigned" },
    })
    invites:create_index("primary", { parts = { "code" }, if_not_exists = true })

    local initial_admin = tonumber(os.getenv("INITIAL_ADMIN_MAX_USER_ID") or "0")
    if initial_admin and initial_admin > 0 then
        organizations:replace({ "system", "System organization", os.time(), true })
        users:replace({ "initial-admin", "system", initial_admin, "Administrator", { "admin" }, "active", os.time(), "" })
    end

    local tasks = box.schema.space.create("tasks", { if_not_exists = true })
    tasks:format({
        { name = "id", type = "string" },
        { name = "organization_id", type = "string" },
        { name = "title", type = "string" },
        { name = "description", type = "string" },
        { name = "object_id", type = "string" },
        { name = "assignee_id", type = "string" },
        { name = "manager_id", type = "string" },
        { name = "status", type = "string" },
        { name = "due_at", type = "number" },
        { name = "created_at", type = "number" },
    })
    tasks:create_index("primary", { parts = { "id" }, if_not_exists = true })
    tasks:create_index("organization_id", { parts = { "organization_id" }, unique = false, if_not_exists = true })
    tasks:create_index("assignee_id", { parts = { "assignee_id" }, unique = false, if_not_exists = true })

    local task_reviews = box.schema.space.create("task_reviews", { if_not_exists = true })
    task_reviews:format({
        { name = "id", type = "string" },
        { name = "task_id", type = "string" },
        { name = "reviewer_id", type = "string" },
        { name = "decision", type = "string" },
        { name = "comment", type = "string" },
        { name = "created_at", type = "number" },
    })
    task_reviews:create_index("primary", { parts = { "id" }, if_not_exists = true })
    task_reviews:create_index("task_id", { parts = { "task_id" }, unique = false, if_not_exists = true })

    local evidence = box.schema.space.create("evidence", { if_not_exists = true })
    evidence:format({
        { name = "id", type = "string" },
        { name = "task_id", type = "string" },
        { name = "kind", type = "string" },
        { name = "object_key", type = "string" },
        { name = "comment", type = "string" },
        { name = "created_at", type = "number" },
    })
    evidence:create_index("primary", { parts = { "id" }, if_not_exists = true })
    evidence:create_index("task_id", { parts = { "task_id" }, unique = false, if_not_exists = true })
end)

box.once("users_manager_id_v1", function()
    local users = box.space.users
    local format = users:format()
    local has_manager_id = false
    for _, field in ipairs(format) do
        if field.name == "manager_id" then
            has_manager_id = true
        end
    end
    if not has_manager_id then
        table.insert(format, { name = "manager_id", type = "string", is_nullable = true })
        users:format(format)
    end
end)
