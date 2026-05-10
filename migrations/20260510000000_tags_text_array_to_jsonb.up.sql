-- Convert tags columns from text[] to jsonb.
-- text[] was created by the original Atlas migration; Ent now maps field.Strings()
-- to jsonb. The explicit USING clause converts native arrays to JSON arrays.
ALTER TABLE "org_units"
    ALTER COLUMN "tags" TYPE jsonb
    USING array_to_json("tags")::jsonb;

ALTER TABLE "users"
    ALTER COLUMN "tags" TYPE jsonb
    USING array_to_json("tags")::jsonb;
