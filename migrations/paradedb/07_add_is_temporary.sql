-- Add is_temporary flag to knowledge_bases to support ephemeral KBs
ALTER TABLE knowledge_bases
  ADD COLUMN IF NOT EXISTS is_temporary BOOLEAN NOT NULL DEFAULT FALSE;


