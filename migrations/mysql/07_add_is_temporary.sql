-- Add is_temporary flag to knowledge_bases to support ephemeral KBs
ALTER TABLE knowledge_bases
  ADD COLUMN IF NOT EXISTS is_temporary TINYINT(1) NOT NULL DEFAULT 0 COMMENT 'Temporary/hidden KB';


