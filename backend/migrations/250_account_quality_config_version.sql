-- Old policies never authorized disclosure to the new external runner.
UPDATE account_quality_policies
SET config = jsonb_set(config, '{enabled}', 'false'::jsonb), next_run_at = NULL, updated_at = NOW()
WHERE config->>'enabled' = 'true'
  AND COALESCE(config->>'candy_consent_version', '') <> 'manxue-v1';

-- Versions contain no credential material. Unversioned historical evidence stays
-- readable, but cannot establish quality for the currently configured upstream.
ALTER TABLE accounts ADD COLUMN quality_config_version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE accounts ADD COLUMN quality_config_changed_at TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01T00:00:00Z';
ALTER TABLE account_quality_results ADD COLUMN config_version BIGINT NOT NULL DEFAULT 0;
ALTER TABLE account_quality_weights ADD COLUMN config_version BIGINT NOT NULL DEFAULT 0;

CREATE FUNCTION account_quality_bump_config_version() RETURNS trigger AS $$
BEGIN
  IF NEW.credentials IS DISTINCT FROM OLD.credentials
     OR NEW.platform IS DISTINCT FROM OLD.platform
     OR NEW.type IS DISTINCT FROM OLD.type
     OR NEW.proxy_id IS DISTINCT FROM OLD.proxy_id
     OR NEW.parent_account_id IS DISTINCT FROM OLD.parent_account_id THEN
    NEW.quality_config_version := OLD.quality_config_version + 1;
    NEW.quality_config_changed_at := clock_timestamp();
    -- Linked shadow accounts inherit the parent's credentials.
    UPDATE accounts SET quality_config_version = quality_config_version + 1,
      quality_config_changed_at = NEW.quality_config_changed_at
    WHERE parent_account_id = OLD.id AND deleted_at IS NULL;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER account_quality_config_changed
BEFORE UPDATE OF credentials, platform, type, proxy_id, parent_account_id ON accounts
FOR EACH ROW EXECUTE FUNCTION account_quality_bump_config_version();

CREATE INDEX account_quality_result_config_scope
ON account_quality_results(account_id, config_version, model, effort, service_tier, endpoint, created_at DESC);
