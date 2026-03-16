-- Agent Conversations (session tracking + audit trail)
CREATE TABLE agent_conversations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         TEXT NOT NULL,
    title           TEXT,
    created_at      TIMESTAMP DEFAULT now(),
    updated_at      TIMESTAMP DEFAULT now(),
    expires_at      TIMESTAMP DEFAULT now() + interval '30 days'
);

-- Agent Messages (display version, depseudonymized)
CREATE TABLE agent_messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES agent_conversations(id) ON DELETE CASCADE,
    role            TEXT NOT NULL,
    content         TEXT NOT NULL,
    created_at      TIMESTAMP DEFAULT now()
);
CREATE INDEX idx_agent_msg_conv ON agent_messages(conversation_id);

-- Agent Memories (persistent cross-session knowledge)
CREATE TABLE agent_memories (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         TEXT NOT NULL,
    memory_type     TEXT NOT NULL,
    content         TEXT NOT NULL,
    metadata        JSONB DEFAULT '{}',
    source          TEXT NOT NULL DEFAULT 'explicit',
    confidence      TEXT DEFAULT 'high',
    created_at      TIMESTAMP DEFAULT now(),
    updated_at      TIMESTAMP DEFAULT now(),
    last_validated  TIMESTAMP DEFAULT now(),
    expires_at      TIMESTAMP
);

-- Agent Audit Log (LGPD Art. 37 — processing records)
CREATE TABLE agent_audit_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         TEXT NOT NULL,
    conversation_id UUID,
    tools_called    TEXT[],
    input_tokens    INT,
    output_tokens   INT,
    provider        TEXT DEFAULT 'vertex_ai',
    region          TEXT DEFAULT 'southamerica-east1',
    created_at      TIMESTAMP DEFAULT now()
);
