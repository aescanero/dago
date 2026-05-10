-- Revert tags columns from jsonb back to text[].
ALTER TABLE "org_units"
    ALTER COLUMN "tags" TYPE text[]
    USING ARRAY(SELECT jsonb_array_elements_text("tags"));

ALTER TABLE "users"
    ALTER COLUMN "tags" TYPE text[]
    USING ARRAY(SELECT jsonb_array_elements_text("tags"));
